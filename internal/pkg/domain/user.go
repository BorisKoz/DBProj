package domain

type User struct {
	Nickname string `json:"nickname"`
	FullName string `json:"fullname"`
	About    string `json:"about"`
	Email    string `json:"email"`
}

type UserRepository interface {
	AddUser(user User) error
	GetUserByNickOrEmail(nickname string, email string) ([]User, error)
	GetUser(nickname string) ([]User, error)
	UpdateUser(user User) (User, error)
}
