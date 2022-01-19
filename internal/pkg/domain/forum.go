package domain

import "time"

type Forum struct {
	Title   string `json:"title"`
	User    string `json:"user"`
	Slug    string `json:"slug"`
	Posts   int64  `json:"posts"`
	Threads int32  `json:"threads"`
}

type Thread struct {
	Id      int32  `json:"id"`
	Title   string `json:"title"`
	Forum   string `json:"forum"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Votes   int32  `json:"votes"`
	Slug    string `json:"slug"`
	Created time.Time `json:"created"`
}

type Post struct {
	Id       int64  `json:"id"`
	Parent   int64  `json:"parent"`
	Author   string `json:"author"`
	Message  string `json:"message"`
	IsEdited bool   `json:"isEdited"`
	Forum    string `json:"forum"`
	Thread   int32  `json:"thread"`
	Created  time.Time `json:"created"`
}

type PostFull struct {
	Post *Post `json:"post,omitempty"`
	Author *User `json:"author,omitempty"`
	Thread *Thread `json:"thread,omitempty"`
	Forum *Forum `json:"forum,omitempty"`
}

type Vote struct {
	Nickname string `json:"nickname"`
	Voice    int32  `json:"voice"`
	IdThread int64  `json:"-"`
}

type Status struct {
	Users int `json:"user"`
	Forums int `json:"forum"`
	Posts int `json:"post"`
	Threads int `json:"thread"`
}

type ForumRepository interface {
	AddForum(forum Forum) (Forum,error)
	GetForum(slug string) (Forum, error)
	GetUsers(slug string, limit int, since string, desc bool) ([]User, error)

	AddThread(thread Thread) (Thread, error)
	GetThreads(slug string, since string, desc bool, limit int) ([]Thread, error)
	CheckThreads(slug string) (bool, error)
	GetThreadIdBySlug(slug string) (int, error)
	AddPosts(id int, posts []Post)  ([]Post, error)
	GetPosts(id int, limit int, since int, sort string, desc bool) ([]Post, error)
	GetThreadInfo(id int) (Thread, error)
	UpdateThread(thread Thread) (Thread, error)

	VoteThread(vote Vote) error
	UpdateVote(vote Vote) error

	GetPost(post Post, related []string) (PostFull, error)
	UpdatePost(post Post) (Post, error)

	ServiceClear() error
	ServiceStatus() (Status, error)


}
