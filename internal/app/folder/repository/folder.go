package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"errors"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/models"
	"github.com/jackc/pgconn"
)

// Коды ошибок PostgreSQL
const UniqueViolation = "23505"

var (
	ErrFolderNotFound = errors.New("folder not found")
	ErrDuplicate      = errors.New("record already exists")
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateFolder(ctx context.Context, folder models.Folder) (int64, error) {
	query := `
		INSERT INTO folders (user_id, name)
		VALUES ($1, $2)
		RETURNING id
	`

	var folderID int64
	err := r.db.QueryRowContext(ctx, query, folder.UserID, folder.Name).Scan(&folderID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				return 0, ErrDuplicate
			}
		}
		return 0, fmt.Errorf("failed to create folder for user %d: %w", folder.UserID, err)
	}
	return folderID, nil
}

func (r *Repository) GetFolderByID(ctx context.Context, folderID int64) (*models.Folder, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM folders
		WHERE id = $1
	`

	var folder models.Folder
	err := r.db.QueryRowContext(ctx, query, folderID).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get folder by id %d: %w", folderID, err)
	}

	return &folder, nil
}

func (r *Repository) UpdateFolderName(ctx context.Context, folderID int64, newName string) error {
	query := `
		UPDATE folders
		SET name = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, newName, folderID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				return ErrDuplicate
			}
		}
		return fmt.Errorf("failed to update folder name for folder %d: %w", folderID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for folder %d: %w", folderID, err)
	}

	if rowsAffected == 0 {
		return ErrFolderNotFound
	}

	return nil
}

func (r *Repository) GetFolderByName(ctx context.Context, userID int64, name string) (*models.Folder, error) {
	query := `
		SELECT id, user_id, name, created_at, updated_at
		FROM folders
		WHERE user_id = $1 AND name = $2
	`

	var folder models.Folder
	err := r.db.QueryRowContext(ctx, query, userID, name).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFolderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get folder by name '%s' for user %d: %w", name, userID, err)
	}

	return &folder, nil
}

func (r *Repository) CountUserFolders(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM folders
		WHERE user_id = $1
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count folders for user %d: %w", userID, err)
	}

	return count, nil
}

func (r *Repository) GetEmailsFromFolder(ctx context.Context, folderID, userID int64, limit, offset int) ([]models.EmailFromFolder, error) {
	query := `
		SELECT
			e.id,
			u.email as sender_email,
			u.name as sender_name,
			u.surname as sender_surname,
			COALESCE(
				(SELECT json_agg(DISTINCT u2.email)
				 FROM user_emails ue2
				 JOIN users u2 ON ue2.user_id = u2.id
				 WHERE ue2.email_id = e.id AND ue2.is_sender = false),
				'[]'::json
			) as receiver_list,
			e.header,
			e.body,
			e.created_at,
			viewer_ue.is_read,
			viewer_ue.is_starred
		FROM folder_emails fe
		JOIN emails e ON fe.email_id = e.id
		JOIN users u ON e.sender_id = u.id
		JOIN LATERAL (
			SELECT is_read, is_starred
			FROM user_emails
			WHERE email_id = e.id AND user_id = $4 AND is_deleted = false
			ORDER BY is_sender ASC
			LIMIT 1
		) viewer_ue ON true
		WHERE fe.folder_id = $1
		ORDER BY fe.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, folderID, limit, offset, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query emails from folder %d: %w", folderID, err)
	}
	defer rows.Close()

	var emails []models.EmailFromFolder
	for rows.Next() {
		var email models.EmailFromFolder
		var receiverListJSON []byte

		err := rows.Scan(
			&email.ID,
			&email.SenderEmail,
			&email.SenderName,
			&email.SenderSurname,
			&receiverListJSON,
			&email.Header,
			&email.Body,
			&email.CreatedAt,
			&email.IsRead,
			&email.IsFavorite,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(receiverListJSON, &email.ReceiverList); err != nil {
			email.ReceiverList = []string{}
		}

		emails = append(emails, email)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

func (r *Repository) CountEmailsInFolder(ctx context.Context, folderID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM folder_emails
		WHERE folder_id = $1
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, folderID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) CountUnreadEmailsInFolder(ctx context.Context, folderID, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM folder_emails fe
		JOIN emails e ON fe.email_id = e.id
		JOIN user_emails ue ON e.id = ue.email_id
			AND ue.user_id = $2
			AND ue.is_deleted = false
		WHERE fe.folder_id = $1 AND ue.is_read = false
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, folderID, userID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_emails ue
			WHERE ue.email_id = $1 AND ue.user_id = $2
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, emailID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// AddEmailsToFolder добавляет несколько писем в папку (батч вставка)
func (r *Repository) AddEmailToFolder(ctx context.Context, folderID, emailID int64) error {
	query := `
        INSERT INTO folder_emails (folder_id, email_id)
        VALUES ($1, $2)
        ON CONFLICT (folder_id, email_id) DO NOTHING
    `

	_, err := r.db.ExecContext(ctx, query, folderID, emailID)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteEmailFromFolder(ctx context.Context, folderID, emailID int64) error {
	query := `
        DELETE FROM folder_emails
        WHERE folder_id = $1 AND email_id = $2
    `

	result, err := r.db.ExecContext(ctx, query, folderID, emailID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return nil
	}

	return nil
}

func (r *Repository) DeleteFolder(ctx context.Context, folderID, userID int64) error {
    query := `
        DELETE FROM folders
        WHERE id = $1 AND user_id = $2
    `

    result, err := r.db.ExecContext(ctx, query, folderID, userID)
    if err != nil {
        return fmt.Errorf("failed to delete folder: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return ErrFolderNotFound
    }

    return nil
}
