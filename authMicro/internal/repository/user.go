package repository

import (
	"context"
	"errors"
)

type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
}

type IUserRepository interface {
	Save(ctx context.Context, user User) error
	FindByEmail(ctx context.Context, email string) (User, error)
}

type UserRepo struct {
	users map[string]User
}

func NewMemoryUserRepo() *UserRepo {
	return &UserRepo{
		users: make(map[string]User),
	}
}

func (m *UserRepo) Save(ctx context.Context, user User) error {
	m.users[user.Email] = user
	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (m *UserRepo) FindByEmail(ctx context.Context, email string) (User, error) {
	user, ok := m.users[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}
