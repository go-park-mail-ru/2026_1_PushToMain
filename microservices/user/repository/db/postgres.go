package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/models"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserDbNotInited = errors.New("user database is not inited")
	ErrQueryError      = errors.New("failed to exec query")
)

type Repository struct {
	userDb *sql.DB
}

func New(userDb *sql.DB) *Repository {
	return &Repository{
		userDb: userDb,
	}
}

func (r *Repository) UpdateProfile(ctx context.Context, userID int64, name, surname string, isMale *bool, birthdate *time.Time) error {
	var setParts []string
	var args []interface{}
	argCounter := 1

	if name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argCounter))
		args = append(args, name)
		argCounter++
	}

	if surname != "" {
		setParts = append(setParts, fmt.Sprintf("surname = $%d", argCounter))
		args = append(args, surname)
		argCounter++
	}

	if isMale != nil {
		setParts = append(setParts, fmt.Sprintf("is_male = $%d", argCounter))
		args = append(args, *isMale)
		argCounter++
	}

	if birthdate != nil {
		setParts = append(setParts, fmt.Sprintf("birthdate = $%d", argCounter))
		args = append(args, *birthdate)
		argCounter++
	}

	if len(setParts) == 0 {
		return nil
	}

	setParts = append(setParts, "updated_at = NOW()")

	query := fmt.Sprintf(`
        UPDATE users
        SET %s
        WHERE id = $%d
    `, strings.Join(setParts, ", "), argCounter)

	args = append(args, userID)

	result, err := r.userDb.ExecContext(ctx, query, args...)
	if err != nil {
		return ErrQueryError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ErrQueryError
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *Repository) UpdateAvatar(ctx context.Context, userID int64, imagePath string) error {
	query := `
        UPDATE users
        SET image_path = $1
        WHERE id = $2
    `

	if r.userDb == nil {
		return ErrUserDbNotInited
	}

	result, err := r.userDb.ExecContext(ctx, query, imagePath, userID)
	if err != nil {
		return ErrQueryError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ErrQueryError
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
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
		return 0, ErrQueryError
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

	user := models.User{Email: email}

	err := repo.userDb.QueryRowContext(ctx, query, email).
		Scan(&user.ID, &user.Password, &user.Name, &user.Surname, &user.ImagePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, ErrQueryError
	}

	return &user, nil
}

func (r *Repository) FindByID(ctx context.Context, userID int64) (*models.User, error) {
	query := `
        SELECT id, email, password_hash, name, surname, image_path, is_male, birthdate
        FROM users
        WHERE id = $1
    `
	if r.userDb == nil {
		return nil, ErrUserDbNotInited
	}

	user := &models.User{}
	err := r.userDb.QueryRowContext(ctx, query, userID).
		Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.Surname, &user.ImagePath, &user.IsMale, &user.Birthdate)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, ErrQueryError
	}

	folders, err := r.GetUserFolders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders for user %d: %w", userID, err)
	}
	user.Folders = folders

	return user, nil
}

func (r *Repository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE id = $2`

	if r.userDb == nil {
		return ErrUserDbNotInited
	}

	result, err := r.userDb.ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return ErrQueryError
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ErrQueryError
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) GetUserFolders(ctx context.Context, userID int64) ([]models.Folder, error) {
	query := `
        SELECT id, name
        FROM folders
        WHERE user_id = $1
        ORDER BY created_at ASC
    `

	rows, err := r.userDb.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query folders: %w", err)
	}
	defer rows.Close()

	var folders []models.Folder
	for rows.Next() {
		var f models.Folder
		if err := rows.Scan(&f.ID, &f.Name); err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}
		folders = append(folders, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return folders, nil
}
