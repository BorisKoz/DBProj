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
)
var (
	UniqueViolation              = "23505"
)

type UserHandler struct {
	ur domain.UserRepository
}

func NewUserHandler(r *router.Router, ur domain.UserRepository) {
	handler := UserHandler{ur: ur}
	r.POST("/api/user/{nickname}/create", handler.Add)
	r.GET("/api/user/{nickname}/profile", handler.Get)
	r.POST("/api/user/{nickname}/profile", handler.Update)
}

func (uh *UserHandler) Add (ctx *fasthttp.RequestCtx) {
	nickname, ok := ctx.UserValue("nickname").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}

	newUser := domain.User{Nickname: nickname}
	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		utils.Send(500, "SE"+err.Error(), ctx)
	}

	err = uh.ur.AddUser(newUser)
	if err != nil {
		users, err := uh.ur.GetUserByNickOrEmail(newUser.Nickname, newUser.Email)
		if err != nil {
			utils.Send(500, "SE"+err.Error(), ctx)
			return
		}
		utils.Send(409, users, ctx)
		return
	}
	utils.Send(201, newUser, ctx)
	return
}


func (uh *UserHandler) Get (ctx *fasthttp.RequestCtx) {
	nickname, ok := ctx.UserValue("nickname").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}

	users, err := uh.ur.GetUser(nickname)
	if err != nil || len(users) == 0 {
		resp := domain.Response{Message: fmt.Sprintf("Can't find user by id %s", nickname)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, users[0], ctx)
	return
}

func (uh *UserHandler) Update (ctx *fasthttp.RequestCtx) {
	nickname, ok := ctx.UserValue("nickname").(string)
	if !ok {
		utils.Send(400, "bad request", ctx)
		return
	}
	newUser := domain.User{Nickname: nickname}
	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		utils.Send(500, "SE"+err.Error(), ctx)
	}

	us, err := uh.ur.UpdateUser(newUser)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				resp := domain.Response{Message: fmt.Sprintf("already in use by %s", nickname)}
				utils.Send(409, resp, ctx)
				return
			}
		}
		resp := domain.Response{Message: fmt.Sprintf("Can't find user by id %s", nickname)}
		utils.Send(404, resp, ctx)
		return
	}
	utils.Send(200, us, ctx)
	return
}