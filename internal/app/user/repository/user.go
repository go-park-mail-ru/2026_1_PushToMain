package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserDbNotInited = errors.New("user database is not inited")

type Repository struct {
	mu    	sync.Mutex
	userDb 	*sql.DB
}

func New(userDb *sql.DB) *Repository {
	return &Repository{
		userDb: userDb,
	}
}

func (repo *Repository) Save(ctx context.Context, user models.User) (int64, error) {
	query := `
		INSERT INTO users (email, password_hash, name, surname, image_path)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	if repo.userDb == nil {
		return 0, ErrUserDbNotInited
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	var userId int64
	err := repo.userDb.QueryRowContext(
		ctx,
		query,
		user.Email,
		user.Password,
		user.Name,
		user.Surname,
		user.ImagePath,
	).Scan(&userId)

	if err != nil {
		return 0, fmt.Errorf("Failed to save user: %w", err)
	}

	return userId, nil
}

func (repo *Repository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, password_hash, name, surname, image_path
		FROM users
		WHERE email = $1
	`

	if repo.userDb == nil {
		return nil, ErrUserDbNotInited
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	user := models.User{Email: email}

	err := repo.userDb.QueryRowContext(ctx, query, email).
        Scan(&user.ID, &user.Password, &user.Name, &user.Surname, &user.ImagePath)
	if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to find user by email: %w", err)
    }

	return &user, nil
}
