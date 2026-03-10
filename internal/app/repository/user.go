package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepo struct {
	mu    sync.Mutex
	users map[string]models.User
}

func NewMemoryUserRepo() *UserRepo {
	return &UserRepo{
		users: make(map[string]models.User),
	}
}

func (repo *UserRepo) Save(ctx context.Context, user models.User) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	repo.users[user.Email] = user
	return nil
}

func (repo *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	user, ok := repo.users[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	return &user, nil
}
