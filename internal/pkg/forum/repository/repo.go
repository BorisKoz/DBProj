package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"repo/internal/pkg/domain"
	"time"
)


type ForumRepository struct {
	dbm *pgxpool.Pool
	userRep domain.UserRepository
}

func NewForumRep (pool *pgxpool.Pool, ur domain.UserRepository) ForumRepository {
	return ForumRepository{dbm: pool, userRep: ur}
}

func (f *ForumRepository) AddForum(forum domain.Forum) (domain.Forum,error) {
	query := "INSERT INTO forum (Title, Usr, Slug) VALUES ($1, $2, $3) RETURNING *;"

	var newForum domain.Forum
	user, err := f.userRep.GetUser(forum.User)
	if err != nil || len(user) == 0 {
		return domain.Forum{}, errors.New("NO SUCH USER")
	}
	row := f.dbm.QueryRow(context.Background(), query, forum.Title, user[0].Nickname, forum.Slug)
	err = row.Scan(&newForum.Title, &newForum.User, &newForum.Slug, &newForum.Posts, &newForum.Threads)
	if err != nil {
		return domain.Forum{}, err
	}
	return newForum, err
}

func (f *ForumRepository) GetForum(slug string) (domain.Forum, error) {
	query := "SELECT Title, Usr, Slug, Posts, Threads from Forum WHERE slug=$1"
	var forum domain.Forum
	row:= f.dbm.QueryRow(context.Background(), query, slug)
	err := row.Scan(&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)
	if err != nil {
		return domain.Forum{}, err
	}
	return forum, err
}

func (f *ForumRepository) GetUsers(slug string, limit int, since string, desc bool) ([]domain.User, error) {
	query := "SELECT u.nickname, u.fullname, u.about, u.email FROM users as u inner join forumUsers as f on u.nickname = f.nickname WHERE f.slug =$1 "
	if desc {
		if since != "" {
			query += fmt.Sprintf(" AND f.nickname < '%s' ", since)
		}
		query += " ORDER BY u.nickname desc "
	} else {
		if since != "" {
			query += fmt.Sprintf(" AND f.nickname > '%s' ", since)
		}
		query +=  " ORDER BY u.nickname "
	}
	query += "LIMIT NULLIF($2, 0)"

	rows, err := f.dbm.Query(context.Background(),query, slug, limit)
	if err != nil {
		return []domain.User{}, err
	}

	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		buffer := domain.User{}
		err = rows.Scan(&buffer.Nickname, &buffer.FullName, &buffer.About, &buffer.Email)
		if err != nil {
			return []domain.User{}, err
		}
		users = append(users, buffer)
	}
	return users, nil
}

func (f *ForumRepository) AddThread(thread domain.Thread) (domain.Thread, error) {
	query := "INSERT INTO Threads (Title, Forum, Message, Author, Slug, Created)  VALUES ($1, $2, $3, $4, $5, $6) RETURNING *"
	newThread := domain.Thread{}
	forum, err := f.GetForum(thread.Forum)
	if err != nil {
		return domain.Thread{}, err
	}
	row := f.dbm.QueryRow(context.Background(), query, thread.Title, forum.Slug, thread.Message, thread.Author, thread.Slug, thread.Created)
	err = row.Scan(&newThread.Id, &newThread.Title, &newThread.Forum, &newThread.Message, &newThread.Author, &newThread.Votes, &newThread.Slug, &newThread.Created)
	if err != nil {
		return domain.Thread{}, err
	}
	return newThread, nil
}

func (f *ForumRepository) GetThreads(slug string, since string, desc bool, limit int) ([]domain.Thread, error) {
	query := "SELECT Id, Title, Forum, Message, Author, Votes, Slug, Created FROM Threads  WHERE forum = $1 "
	if desc {
		if since != "" {
			query += fmt.Sprintf(" AND created <= '%s' ", since)
		}
		query += " ORDER BY created desc "
	} else {
		if since != "" {
			query += fmt.Sprintf(" AND created >= '%s' ", since)
		}
		query += " ORDER BY created asc "
	}
	query += " LIMIT NULLIF($2, 0)"
	rows, err := f.dbm.Query(context.Background(),query, slug, limit)
	if err != nil {
		return []domain.Thread{}, err
	}

	defer rows.Close()
	threads := []domain.Thread{}
	for rows.Next() {
		newThread := domain.Thread{}
		err = rows.Scan(&newThread.Id, &newThread.Title, &newThread.Forum, &newThread.Message, &newThread.Author, &newThread.Votes, &newThread.Slug, &newThread.Created)
		if err != nil {
			return []domain.Thread{}, err
		}
		threads = append(threads, newThread)
	}
	return threads, nil
}

func (f *ForumRepository) CheckThreads(slug string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM Threads WHERE forum=$1)"
	notNull := false
	err := f.dbm.QueryRow(context.Background(), query, slug).Scan(&notNull)
	return notNull, err
}

func (f *ForumRepository) GetThreadIdBySlug(slug string) (int, error) {
	query := "SELECT Id FROM Threads WHERE slug = $1"
	row := f.dbm.QueryRow(context.Background(), query, slug)
	newThread := domain.Thread{}
	err := row.Scan(&newThread.Id)
	if err != nil {
		return -1, err
	}
	return int(newThread.Id), err
}

func (f *ForumRepository) AddPosts(id int, posts []domain.Post) ([]domain.Post, error) {
	query := "INSERT INTO Posts (Parent, Author, Message, IsEdited, Forum, Thread, Created) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING Id, Parent, Author, Message, IsEdited, Forum, Thread, Created"
	newPosts := []domain.Post{}
	thread, err := f.GetThreadInfo(id)
	if err != nil {
		fmt.Println(err)
		return []domain.Post{},err
	}
	createdTime := time.Now().Format(time.RFC3339)
	for _,post := range posts {
		newPost := domain.Post{}
		row := f.dbm.QueryRow(context.Background(), query, post.Parent, post.Author, post.Message, post.IsEdited,
			thread.Forum, id, createdTime)
		err := row.Scan(&newPost.Id, &newPost.Parent, &newPost.Author, &newPost.Message, &newPost.IsEdited,
			&newPost.Forum, &newPost.Thread, &newPost.Created)
		if err != nil {
			return newPosts, err
		}
		newPosts = append(newPosts, newPost)
	}
	return newPosts, nil
}

func (f *ForumRepository) GetPosts(id int, limit int, since int, sort string, desc bool) ([]domain.Post, error) {
	switch sort {
	case "flat", "":
		query:= "SELECT Id, Parent, Author, Message, IsEdited, Forum, Thread, Created FROM Posts WHERE Thread = $1"
		if desc {
			if since > 0 {
				query += fmt.Sprintf(" AND id < %d ", since)
			}
			query += " ORDER BY id DESC "
		} else {
			if since > 0 {
				query += fmt.Sprintf(" AND id > %d ", since)
			}
			query += " ORDER BY id "
		}
		query += " LIMIT NULLIF($2, 0)"
		posts := []domain.Post{}
		rows, err  := f.dbm.Query(context.Background(), query, id, limit)
		defer rows.Close()
		if err != nil {
			return posts, err
		}
		for rows.Next() {
			gotten := domain.Post{}
			err = rows.Scan(&gotten.Id, &gotten.Parent, &gotten.Author, &gotten.Message, &gotten.IsEdited, &gotten.Forum, &gotten.Thread, &gotten.Created)
			if err!=nil {
				return posts, err
			}
			posts = append(posts, gotten)
		}
		return posts, nil
	case "tree":
		query := "SELECT Id, Parent, Author, Message, IsEdited, Forum, Thread, Created FROM Posts WHERE thread=$1 "
		if desc {
			if since > 0 {
				query += fmt.Sprintf(" AND treeOrder < (SELECT treeOrder FROM Posts WHERE id = %d)", since)
			}
			query += " ORDER BY treeOrder DESC, id DESC"
		} else {
			if since > 0 {
				query += fmt.Sprintf(" AND treeOrder > (SELECT treeOrder FROM Posts WHERE id = %d)", since)
			}
			query += " ORDER BY treeOrder, id "
		}
		if limit > 0 {
			query += " LIMIT $2"
		}
		posts := []domain.Post{}
		rows, err  := f.dbm.Query(context.Background(), query, id, limit)
		defer rows.Close()
		if err != nil {
			return posts, err
		}
		for rows.Next() {
			gotten := domain.Post{}
			err = rows.Scan(&gotten.Id, &gotten.Parent, &gotten.Author, &gotten.Message, &gotten.IsEdited, &gotten.Forum, &gotten.Thread, &gotten.Created)
			posts = append(posts, gotten)
		}
		return posts, nil
	case "parent_tree":
		query := "SELECT Id, Parent, Author, Message, IsEdited, Forum, Thread, Created FROM Posts WHERE "
		query += " treeOrder[1] IN (SELECT ID FROM Posts WHERE Thread =$1 AND Parent=0 "
		if desc {
			if since > 0 {
				query += fmt.Sprintf(" AND treeOrder[1] < (SELECT treeOrder[1] FROM Posts WHERE id = %d) ", since)
			}
			query += " ORDER BY Id DESC"
			if limit > 0 {
				query += " LIMIT $2 "
			}
			query += " ) ORDER BY treeOrder[1] DESC, treeOrder, id"
		} else {
			if since > 0 {
				query += fmt.Sprintf("AND treeOrder[1] > (SELECT treeOrder[1] FROM Posts WHERE id = %d) ", since)
			}
			query += " ORDER BY Id"
			if limit > 0 {
				query += " LIMIT $2 "
			}
			query += " ) ORDER BY treeOrder[1],treeOrder, id "
		}
		posts := []domain.Post{}
		rows, err  := f.dbm.Query(context.Background(), query, id, limit)
		defer rows.Close()
		if err != nil {
			return posts, err
		}
		for rows.Next() {
			gotten := domain.Post{}
			err = rows.Scan(&gotten.Id, &gotten.Parent, &gotten.Author, &gotten.Message, &gotten.IsEdited, &gotten.Forum, &gotten.Thread, &gotten.Created)
			posts = append(posts, gotten)
		}
		return posts, nil
	default:
		return nil, errors.New("NoSort")
	}
}

func (f *ForumRepository) GetThreadInfo(id int) (domain.Thread, error) {
	query := "SELECT Id, Title, Forum, Message, Author, Votes, Slug, Created  FROM threads Where ID = $1"
	rows := f.dbm.QueryRow(context.Background(),query, id)
	newThread := domain.Thread{}
	err := rows.Scan(&newThread.Id, &newThread.Title, &newThread.Forum, &newThread.Message, &newThread.Author, &newThread.Votes, &newThread.Slug, &newThread.Created)
	if err != nil {
		return domain.Thread{}, err
	}
	return newThread, err

}

func (f *ForumRepository) UpdateThread(thread domain.Thread) (domain.Thread, error) {
	query := "UPDATE threads SET Title = COALESCE(NULLIF($1, ''), Title), " +
		" Message = COALESCE(NULLIF($2, ''), Message) WHERE id = $3 RETURNING *"
	rows := f.dbm.QueryRow(context.Background(),query, thread.Title, thread.Message, thread.Id)
	newThread := domain.Thread{}
	err := rows.Scan(&newThread.Id, &newThread.Title, &newThread.Forum, &newThread.Message, &newThread.Author, &newThread.Votes, &newThread.Slug, &newThread.Created)
	if err != nil {
		return domain.Thread{}, err
	}
	return newThread, err
}

func (f *ForumRepository) VoteThread(vote domain.Vote) error {
	query := "INSERT INTO Votes (Nickname, Voice, IdThread) VALUES ($1, $2, $3)"
	_, err := f.dbm.Exec(context.Background(),query, vote.Nickname, vote.Voice, vote.IdThread)
	return err
}
func (f *ForumRepository) UpdateVote(vote domain.Vote) error {
	query:= "UPDATE Votes SET Voice = $1 WHERE IdThread = $2 AND Nickname = $3"
	_, err := f.dbm.Exec(context.Background(),query, vote.Voice, vote.IdThread, vote.Nickname)
	return err
}

func (f *ForumRepository) GetPost(post domain.Post, related []string) (domain.PostFull, error) {
	query:= "SELECT Id, Parent, Author, Message, IsEdited, Forum, Thread, Created from Posts WHERE id = $1"
	row :=  f.dbm.QueryRow(context.Background(), query, post.Id)
	gotten := domain.Post{}
	err := row.Scan(&gotten.Id, &gotten.Parent, &gotten.Author, &gotten.Message, &gotten.IsEdited, &gotten.Forum, &gotten.Thread, &gotten.Created)
	if err != nil {
		return domain.PostFull{}, err
	}
	result := domain.PostFull{Post: &gotten}
	
	for _, relType := range related {
		if relType == "user" {
			us, err := f.userRep.GetUser(gotten.Author)
			if err != nil {
				return result, err
			}
			result.Author = &us[0]
		} else if relType == "forum" {
			fr, err := f.GetForum(gotten.Forum)
			if err != nil {
				return result, err
			}
			result.Forum = &fr
		} else if relType == "thread" {
			th, err := f.GetThreadInfo(int(gotten.Thread))
			if err != nil {
				return result, err
			}
			result.Thread = &th
		}
	}
	return result, nil
}

func (f *ForumRepository) UpdatePost(post domain.Post) (domain.Post, error) {
	old, err := f.GetPost(domain.Post{Id:post.Id}, []string{})
	if err != nil {
		return domain.Post{}, err
	}
	if old.Post.Message == post.Message || post.Message == "" {
		return *old.Post, err
	}
	query := "UPDATE Posts SET message = $1, isEdited = true WHERE id = $2 RETURNING Id, Parent, Author, Message, IsEdited, Forum, Thread, Created"
	row :=  f.dbm.QueryRow(context.Background(), query, post.Message, post.Id)
	gotten := domain.Post{}
	err = row.Scan(&gotten.Id, &gotten.Parent, &gotten.Author, &gotten.Message, &gotten.IsEdited, &gotten.Forum, &gotten.Thread, &gotten.Created)
	if err != nil {
		return domain.Post{}, err
	}
	return gotten, nil
}

func (f *ForumRepository) ServiceClear() error {
	query := `TRUNCATE Users, Forum, Threads, Posts, Votes, forumUsers`
	_, err := f.dbm.Exec(context.Background(),query)
	return err
}

func (f *ForumRepository) ServiceStatus() (domain.Status,error) {
	query := "SELECT * FROM (SELECT COUNT(*) FROM Forum) as forumCount, (SELECT COUNT(*) FROM Threads) as threadCount, (SELECT COUNT(*) FROM Users) as userCount, (SELECT COUNT(*) FROM Posts) as postCount"
	row := f.dbm.QueryRow(context.Background(), query)
	st := domain.Status{}
	err := row.Scan(&st.Forums, &st.Threads, &st.Users, &st.Posts)
	if err != nil {
		return domain.Status{}, err
	}
	return st, err
}


