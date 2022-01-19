package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/jackc/pgconn"
	"github.com/valyala/fasthttp"
	"repo/internal/pkg/domain"
	"repo/internal/pkg/utils"
	"strconv"
	"strings"
)

var (
	ForeignKeyViolation          = "23503"
	UniqueViolation              = "23505"
)

type ForumHandler struct {
	fr domain.ForumRepository
}

func NewForumHandler(r *router.Router, fr domain.ForumRepository) {
	handler := ForumHandler{fr: fr}
	// forum funcs
	r.POST("/api/forum/create", handler.AddForum)
	r.GET("/api/forum/{slug}/details", handler.GetForum)
	r.GET("/api/forum/{slug}/users", handler.GetUsers)

	// thread funcs
	r.POST("/api/forum/{slug}/create", handler.AddThread)
	r.GET("/api/forum/{slug}/threads", handler.GetThreads)
	r.GET("/api/thread/{slug_or_id}/details", handler.GetThread)
	r.POST("/api/thread/{slug_or_id}/details", handler.UpdateThread)

	// post funcs
	r.POST("/api/thread/{slug_or_id}/create", handler.AddPosts)
	r.GET("/api/thread/{slug_or_id}/posts", handler.GetPosts)
	r.GET("/api/post/{id:[0-9]+}/details", handler.GetPost)
	r.POST("/api/post/{id:[0-9]+}/details", handler.UpdatePost)

	// vote funcs
	r.POST("/api/thread/{slug_or_id}/vote", handler.VoteThread)

	// service funcs
	r.GET("/api/service/status", handler.Status)
	r.POST("/api/service/clear", handler.Clear)
}

func (fh *ForumHandler) AddForum (ctx *fasthttp.RequestCtx) {
	forum := domain.Forum{}
	err := json.Unmarshal(ctx.PostBody(), &forum)
	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	fr, err := fh.fr.AddForum(forum)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation  {
				old, _ := fh.fr.GetForum(forum.Slug)
				utils.Send(409, old, ctx)
				return
			}
		}
		resp := domain.Response{Message: fmt.Sprintf("Can't find user by id %s", forum.User)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(201, fr, ctx)
	return
}

func (fh *ForumHandler) GetForum (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	fr, err := fh.fr.GetForum(slug)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("Can't find forum with slug: %s", slug)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, fr, ctx)
	return
}

func (fh *ForumHandler) GetUsers (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	limit, err := utils.GetQueryInt(ctx, "limit")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	since := utils.GetQueryString(ctx, "since")
	desc, err := utils.GetQueryBool(ctx, "desc")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	_, err = fh.fr.GetForum(slug)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("Can't find forum with slug: %s", slug)}
		utils.Send(404, resp, ctx)
		return
	}
	fr, err := fh.fr.GetUsers(slug, limit, since, desc)
	if err != nil {
		utils.Send(200, make([]domain.User,0,0), ctx)
		return
	}
	utils.Send(200, fr, ctx)
	return
}

func (fh *ForumHandler) AddThread (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	thread := domain.Thread{Forum: slug}
	err := json.Unmarshal(ctx.PostBody(), &thread)
	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	th, err := fh.fr.AddThread(thread)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation  {
				id, _ := fh.fr.GetThreadIdBySlug(thread.Slug)
				old, _ := fh.fr.GetThreadInfo(id)
				utils.Send(409, old, ctx)
				return
			}
		}
		resp := domain.Response{Message: fmt.Sprintf("Can't find user by id %s", thread.Author)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(201, th, ctx)
}

func (fh *ForumHandler) GetThread (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug_or_id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		id, _ = fh.fr.GetThreadIdBySlug(slug)
	}
	th, err := fh.fr.GetThreadInfo(id)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("Can't find thread with slug or id: %s", slug)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, th, ctx)
	return
}

func (fh *ForumHandler) GetThreads (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	since := utils.GetQueryString(ctx, "since")
	desc, err := utils.GetQueryBool(ctx, "desc")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	limit, err := utils.GetQueryInt(ctx, "limit")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	thrs, err := fh.fr.GetThreads(slug, since, desc, limit)
	if err != nil || len(thrs) == 0 {
		// as len(thrs) == 0 can indicate both empty result and no forum threads - check existence
		notNull, _ := fh.fr.CheckThreads(slug)
		if notNull {
			utils.Send(200, make([]domain.Thread, 0, 0),ctx)
			return
		}
		resp := domain.Response{Message: fmt.Sprintf("Can't find threads of forum: %s", slug)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, thrs, ctx)
	return
}

func (fh *ForumHandler) UpdateThread (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug_or_id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		id, _ = fh.fr.GetThreadIdBySlug(slug)
	}
	thread := domain.Thread{Id: int32(id)}
	err = json.Unmarshal(ctx.PostBody(), &thread)
	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	th, err := fh.fr.UpdateThread(thread)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("Can't find threads of forum: %s", slug)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, th, ctx)
	return
}

func (fh *ForumHandler) AddPosts (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug_or_id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		id, _= fh.fr.GetThreadIdBySlug(slug)
	}
	_, err = fh.fr.GetThreadInfo(id)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("thread of id is missing %d", id)}
		utils.Send(404, resp, ctx)
		return
	}

	posts := []domain.Post{}
	err = json.Unmarshal(ctx.PostBody(), &posts)
	if len(posts) == 0 || posts == nil {
		utils.Send(201, []domain.Post{}, ctx)
		return
	}

	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	ps, err := fh.fr.AddPosts(id, posts)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				resp := domain.Response{Message: fmt.Sprintf("Parent is absent")}
				utils.Send(409, resp, ctx)
				return
			}
		}
		resp := domain.Response{Message: fmt.Sprintf("thread of id is missing %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(201, ps, ctx)
	return
}

func (fh *ForumHandler) GetPosts (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug_or_id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		id, err = fh.fr.GetThreadIdBySlug(slug)
		if err != nil {
			resp := domain.Response{Message: fmt.Sprintf("No thread of id %d", id)}
			utils.Send(404, resp, ctx)
			return
		}
	}
	_, err = fh.fr.GetThreadInfo(id)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("No thread of id %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	limit, err := utils.GetQueryInt(ctx, "limit")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	since, err := utils.GetQueryInt(ctx, "since")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	sort := utils.GetQueryString(ctx, "sort")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	desc, err := utils.GetQueryBool(ctx, "desc")
	if err != nil {
		utils.Send(400, "bad request", ctx)
		return
	}
	posts, err := fh.fr.GetPosts(id, limit, since, sort, desc)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("No thread of id %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, posts, ctx)
	return
}

func (fh *ForumHandler) GetPost (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		return
	}
	related := string(ctx.QueryArgs().Peek("related"))
	post, err := fh.fr.GetPost(domain.Post{Id: int64(id)}, strings.Split(related, ","))
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("No post of id %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200,post, ctx)
	return
}

func (fh *ForumHandler) UpdatePost (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		return
	}
	post := domain.Post{Id:int64(id)}
	err = json.Unmarshal(ctx.PostBody(), &post)
	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	edit, err := fh.fr.UpdatePost(post)
	if err != nil {
		resp := domain.Response{Message: fmt.Sprintf("No post of id %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, edit, ctx)
	return
}

func (fh *ForumHandler) VoteThread (ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug_or_id").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	id, err := strconv.Atoi(slug)
	if err != nil {
		id, _ = fh.fr.GetThreadIdBySlug(slug)
	}
	vote := domain.Vote{IdThread: int64(id)}
	err = json.Unmarshal(ctx.PostBody(), &vote)
	if err != nil {
		utils.Send(500, err.Error(), ctx)
		return
	}
	err = fh.fr.VoteThread(vote)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				err = fh.fr.UpdateVote(vote)
				th, _ := fh.fr.GetThreadInfo(id)
				utils.Send(200, th, ctx)
				return
			}
		}
		resp := domain.Response{Message: fmt.Sprintf("No thread of id %d", id)}
		utils.Send(404, resp, ctx)
		return
	}
	th, err := fh.fr.GetThreadInfo(id)
	utils.Send(200, th, ctx)
	return
}

func (fh *ForumHandler) Status (ctx *fasthttp.RequestCtx) {
	info, err := fh.fr.ServiceStatus()
	if err != nil {
		utils.Send(404, err.Error(), ctx)
		return
	}
	utils.Send(200, info, ctx)
	return
}

func (fh *ForumHandler) Clear (ctx *fasthttp.RequestCtx) {
	err := fh.fr.ServiceClear()
	if err != nil {
		utils.Send(404, err.Error(), ctx)
		return
	}
	utils.Send(200, "done", ctx)
	return
}