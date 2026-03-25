package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/models"
)

var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	mu    sync.Mutex
	users map[string]models.User
}

func New() *Repository {
	return &Repository{
		users: make(map[string]models.User),
	}
}

func (repo *Repository) Save(ctx context.Context, user models.User) (int64, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	user.ID = 1 //todo пока не прикрутили реалую бд просто id 1 будет выдавать
	repo.users[user.Email] = user
	return user.ID, nil
}

func (repo *Repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	user, ok := repo.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	return &user, nil
}
