package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUsersByEmails(ctx context.Context, emails []string) (map[string]int64, error) {
	if len(emails) == 0 {
		return make(map[string]int64), nil
	}

	placeholders := make([]string, len(emails))
	args := make([]interface{}, len(emails))
	for i, email := range emails {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = email
	}

	query := fmt.Sprintf(`
		SELECT email, id 
		FROM users 
		WHERE email IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users by emails: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var email string
		var userID int64
		if err := rows.Scan(&email, &userID); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		result[email] = userID
	}

	return result, nil
}

func (r *Repository) SaveEmail(ctx context.Context, email models.Email) (int64, error) {
	query := `
		INSERT INTO emails (sender_id, header, body, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`

	var emailID int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		email.SenderID,
		email.Header,
		email.Body,
	).Scan(&emailID)

	if err != nil {
		return 0, fmt.Errorf("failed to save email: %w", err)
	}

	return emailID, nil
}

func (r *Repository) AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error {
	if len(receiverIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO user_emails (receiver_id, email_id, is_read, created_at, updated_at)
		VALUES ($1, $2, false, NOW(), NOW())
	`

	for _, receiverID := range receiverIDs {
		_, err = tx.ExecContext(ctx, query, receiverID, emailID)
		if err != nil {
			return fmt.Errorf("failed to add receiver %d: %w", receiverID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error) {
	query := `
		SELECT 
			e.id,
			e.sender_id,
			e.header,
			e.body,
			e.created_at,
			ue.is_read,
			ue.created_at as received_at
		FROM emails e
		JOIN user_emails ue ON e.id = ue.email_id
		WHERE ue.receiver_id = $1
		ORDER BY ue.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var email models.Email
		var isRead bool
		var receivedAt sql.NullTime

		err := rows.Scan(
			&email.ID,
			&email.SenderID,
			&email.Header,
			&email.Body,
			&email.CreatedAt,
			&isRead,
			&receivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email: %w", err)
		}

		emails = append(emails, email)
	}

	return emails, nil
}
