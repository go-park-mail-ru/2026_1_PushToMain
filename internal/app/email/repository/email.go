package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/lib/pq"
)

// Коды ошибок PostgreSQL
const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrQueryFail         = errors.New("failed to find mails")
	ErrMailNotFound      = errors.New("emails not found")
	ErrTransactionFailed = errors.New("transaction failed")
	ErrSaveEmail         = errors.New("failed to save email")
	ErrReceiverAdd       = errors.New("failed to add receivers")
	ErrDuplicate         = errors.New("record already exists")
	ErrForeignKey        = errors.New("related record not found")
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error) {
	if len(emails) == 0 {
		return []*models.User{}, nil
	}

	placeholders := make([]string, len(emails))
	args := make([]interface{}, len(emails))
	for i, email := range emails {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = email
	}

	query := fmt.Sprintf(`
		SELECT id, email, name, surname
		FROM users 
		WHERE email IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Surname); err != nil {
			return nil, ErrQueryFail
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) SaveEmail(ctx context.Context, email models.Email) (int64, error) {
	query := `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
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
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == UniqueViolation {
				return 0, ErrDuplicate
			}
			if pqErr.Code == ForeignKeyViolation {
				return 0, ErrForeignKey
			}
		}
		return 0, ErrSaveEmail
	}

	return emailID, nil
}

func (r *Repository) AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error {
	if len(receiverIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(receiverIDs))
	args := make([]interface{}, 0, len(receiverIDs)*2)

	for i, receiverID := range receiverIDs {
		placeholders[i] = fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, receiverID, emailID)
	}

	query := fmt.Sprintf(`
		INSERT INTO user_emails (receiver_id, email_id)
		VALUES %s
	`, strings.Join(placeholders, ", "))

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == UniqueViolation {
				return ErrDuplicate
			}
			if pqErr.Code == ForeignKeyViolation {
				return ErrForeignKey
			}
		}
		return ErrReceiverAdd
	}

	return nil
}

func (r *Repository) SaveEmailWithTx(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error) {
	query := `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var emailID int64
	err := tx.QueryRowContext(
		ctx,
		query,
		email.SenderID,
		email.Header,
		email.Body,
	).Scan(&emailID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == UniqueViolation {
				return 0, ErrDuplicate
			}
			if pqErr.Code == ForeignKeyViolation {
				return 0, ErrForeignKey
			}
		}
		return 0, ErrSaveEmail
	}

	return emailID, nil
}

func (r *Repository) AddEmailReceiversWithTx(ctx context.Context, tx *sql.Tx, emailID int64, receiverIDs []int64) error {
	if len(receiverIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(receiverIDs))
	args := make([]interface{}, 0, len(receiverIDs)*2)

	for i, receiverID := range receiverIDs {
		placeholders[i] = fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, receiverID, emailID)
	}

	query := fmt.Sprintf(`
		INSERT INTO user_emails (receiver_id, email_id)
		VALUES %s
	`, strings.Join(placeholders, ", "))

	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == UniqueViolation {
				return ErrDuplicate
			}
			if pqErr.Code == ForeignKeyViolation {
				return ErrForeignKey
			}
		}
		return ErrReceiverAdd
	}

	return nil
}

func (r *Repository) GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

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
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, checkError(err)
	}
	defer rows.Close()

	emails := make([]models.EmailWithMetadata, 0)
	for rows.Next() {
		var email models.EmailWithMetadata

		err := rows.Scan(
			&email.ID,
			&email.SenderID,
			&email.Header,
			&email.Body,
			&email.CreatedAt,
			&email.IsRead,
		)
		if err != nil {
			return nil, ErrQueryFail
		}

		emails = append(emails, email)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

func (r *Repository) GetEmailByID(ctx context.Context, emailID int64) (*models.Email, error) {
	query := `
        SELECT id, sender_id, header, body, created_at
        FROM emails
        WHERE id = $1
    `

	var email models.Email
	err := r.db.QueryRowContext(ctx, query, emailID).Scan(
		&email.ID,
		&email.SenderID,
		&email.Header,
		&email.Body,
		&email.CreatedAt,
	)

	if err != nil {
		return nil, checkError(err)
	}

	return &email, nil
}

func (r *Repository) GetEmailsCount(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_emails
		WHERE receiver_id = $1
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, checkError(err)
	}

	return count, nil
}

func (r *Repository) MarkEmailAsRead(ctx context.Context, emailID, userID int64) error {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_emails WHERE email_id = $1 AND receiver_id = $2)`
	err := r.db.QueryRowContext(ctx, checkQuery, emailID, userID).Scan(&exists)
	if err != nil {
		return ErrQueryFail
	}
	if !exists {
		return ErrMailNotFound
	}

	query := `
		UPDATE user_emails
		SET is_read = true, updated_at = NOW()
		WHERE email_id = $1 AND receiver_id = $2 AND is_read = false
	`

	_, err = r.db.ExecContext(ctx, query, emailID, userID)
	if err != nil {
		return checkError(err)
	}

	return nil
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM emails e
			LEFT JOIN user_emails ue ON e.id = ue.email_id
			WHERE e.id = $1 
			AND (e.sender_id = $2 OR ue.receiver_id = $2)
		)
	`

	var hasAccess bool
	err := r.db.QueryRowContext(ctx, query, emailID, userID).Scan(&hasAccess)
	if err != nil {
		return false, fmt.Errorf("failed to check email access: %w", err)
	}

	return hasAccess, nil
}

func checkError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrMailNotFound
	}
	return ErrQueryFail
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}
