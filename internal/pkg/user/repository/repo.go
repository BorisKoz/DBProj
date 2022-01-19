package repository

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"repo/internal/pkg/domain"
)


type UserRepository struct {
	dbm *pgxpool.Pool
}

func NewUserRep (pool *pgxpool.Pool) UserRepository {
	return UserRepository{dbm: pool}
}

func (ur *UserRepository) AddUser(user domain.User) error {
	query := "INSERT INTO users (nickname, fullname, about, email) VALUES ($1, $2, $3, $4)"
	_, err := ur.dbm.Exec(context.Background(), query, user.Nickname, user.FullName, user.About, user.Email)
	return err
}

func (ur *UserRepository) GetUserByNickOrEmail(nickname string, email string) ([]domain.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1) OR Email=$2`

	var rows []domain.User
	row, err := ur.dbm.Query(context.Background(), query, nickname, email)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	for row.Next() {
		var user domain.User
		err = row.Scan(&user.Nickname, &user.FullName, &user.About, &user.Email)
		if err != nil {
			return nil, err
		}
		rows = append(rows, user)
	}
	return rows, err
}

func (ur *UserRepository) GetUser(nickname string) ([]domain.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1)`

	var rows []domain.User
	row, err := ur.dbm.Query(context.Background(), query, nickname)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	for row.Next() {
		var user domain.User
		err = row.Scan(&user.Nickname, &user.FullName, &user.About, &user.Email)
		if err != nil {
			return nil, err
		}
		rows = append(rows, user)
	}
	return rows, err
}

func (ur *UserRepository) UpdateUser(user domain.User) (domain.User, error) {
	query := "UPDATE users SET FullName = COALESCE(NULLIF($1, ''), FullName), About = COALESCE(NULLIF($2, ''), About), Email = COALESCE(NULLIF($3, ''), Email) WHERE LOWER(nickname) = LOWER($4) RETURNING *"
	row:= ur.dbm.QueryRow(context.Background(), query, user.FullName, user.About, user.Email, user.Nickname)
	us:= domain.User{Nickname: user.Nickname}
	err := row.Scan(&us.Nickname, &us.FullName, &us.About, &us.Email)
	if err != nil {
		return domain.User{}, err
	}
	return us, nil
}
