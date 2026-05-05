package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

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
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Surname); err != nil {
			return nil, ErrQueryFail
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *Repository) SaveEmail(ctx context.Context, email models.Email) (int64, error) {
	const query = `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id int64
	if err := r.db.QueryRowContext(ctx, query, email.SenderID, email.Header, email.Body).Scan(&id); err != nil {
		return 0, mapPgError(err)
	}
	return id, nil
}

func (r *Repository) SaveEmailWithTx(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error) {
	const query = `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id int64
	if err := tx.QueryRowContext(ctx, query, email.SenderID, email.Header, email.Body).Scan(&id); err != nil {
		return 0, mapPgError(err)
	}
	return id, nil
}

func (r *Repository) AddEmailUserWithTx(ctx context.Context, tx *sql.Tx, emailID, userID int64, isSender bool) error {
	if isSender {
		const q = `
			INSERT INTO user_emails (user_id, email_id, is_sender, is_spam)
			VALUES ($1, $2, true, false)
		`
		if _, err := tx.ExecContext(ctx, q, userID, emailID); err != nil {
			return mapPgErrorReceiver(err)
		}
		return nil
	}

	const q = `
		INSERT INTO user_emails (user_id, email_id, is_sender, is_spam)
		SELECT $1, $2, false, EXISTS(
			SELECT 1
			FROM spam_senders ss
			JOIN emails e ON e.id = $2
			WHERE ss.user_id = $1 AND ss.sender_id = e.sender_id
		)
	`
	if _, err := tx.ExecContext(ctx, q, userID, emailID); err != nil {
		return mapPgErrorReceiver(err)
	}
	return nil
}

func (r *Repository) CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM user_emails WHERE email_id = $1 AND user_id = $2)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, emailID, userID).Scan(&exists); err != nil {
		return false, ErrQueryFail
	}
	return exists, nil
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) error {
	exists, err := r.CheckUserEmailExists(ctx, emailID, userID)
	if err != nil || !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *Repository) GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error) {
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			COALESCE(u.image_path, '') AS image_path,
			COALESCE((
				SELECT array_agg(ru.email ORDER BY ru.id)
				FROM user_emails rue
				JOIN users ru ON rue.user_id = ru.id
				WHERE rue.email_id = e.id AND rue.is_sender = false
			), '{}'::text[]) AS receivers_emails
		FROM emails e
		JOIN users u ON e.sender_id = u.id
		WHERE e.id = $1
	`
	var em models.EmailWithAvatar
	var receiversStr string
	err := r.db.QueryRowContext(ctx, query, emailID).Scan(
		&em.ID, &em.SenderID, &em.Header, &em.Body, &em.CreatedAt,
		&em.SenderImagePath, &receiversStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMailNotFound
		}
		return nil, ErrQueryFail
	}
	em.ReceiversEmails = parsePgTextArray(receiversStr)
	return &em, nil
}

func (r *Repository) GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
			ue.created_at AS received_at,
			COALESCE((
				SELECT array_agg(ru.email ORDER BY ru.id)
				FROM user_emails rue
				JOIN users ru ON rue.user_id = ru.id
				WHERE rue.email_id = e.id AND rue.is_sender = false
			), '{}'::text[]) AS receivers_emails
		FROM user_emails ue
		JOIN emails e ON ue.email_id = e.id
		WHERE ue.user_id = $1
		  AND ue.is_sender = false
		  AND ue.is_deleted = false
		  AND ue.is_spam = false
		  AND ue.is_draft = false
		ORDER BY ue.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			false AS is_read, ue.is_starred, false AS is_spam, ue.is_deleted,
			ue.created_at AS received_at,
			COALESCE((
				SELECT array_agg(ru.email ORDER BY ru.id)
				FROM user_emails rue
				JOIN users ru ON rue.user_id = ru.id
				WHERE rue.email_id = e.id AND rue.is_sender = false
			), '{}'::text[]) AS receivers_emails
		FROM user_emails ue
		JOIN emails e ON ue.email_id = e.id
		WHERE ue.user_id = $1
		  AND ue.is_sender = true
		  AND ue.is_deleted = false
		  AND ue.is_draft = false
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) queryEmailsList(ctx context.Context, query string, args ...interface{}) ([]models.EmailWithMetadata, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	emails := make([]models.EmailWithMetadata, 0)
	for rows.Next() {
		var em models.EmailWithMetadata
		var receiversStr string
		if err := rows.Scan(
			&em.ID, &em.SenderID, &em.Header, &em.Body, &em.CreatedAt,
			&em.IsRead, &em.IsStarred, &em.IsSpam, &em.IsDeleted,
			&em.ReceivedAt, &receiversStr,
		); err != nil {
			return nil, ErrQueryFail
		}
		em.ReceiversEmails = parsePgTextArray(receiversStr)
		emails = append(emails, em)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return emails, nil
}

func (r *Repository) scanCount(ctx context.Context, query string, args ...interface{}) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, ErrQueryFail
	}
	return count, nil
}

func (r *Repository) GetEmailsCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_deleted = false AND is_spam = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false AND is_read = false
		  AND is_deleted = false AND is_spam = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) GetSenderEmailsCount(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = true
		  AND is_deleted = false AND is_draft = false
	`
	return r.scanCount(ctx, query, userID)
}

func (r *Repository) MarkEmailAsRead(ctx context.Context, emailID, userID int64) error {
	return r.toggleReadFlag(ctx, emailID, userID, true)
}

func (r *Repository) MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error {
	return r.toggleReadFlag(ctx, emailID, userID, false)
}

func (r *Repository) toggleReadFlag(ctx context.Context, emailID, userID int64, read bool) error {
	const query = `
		UPDATE user_emails
		SET is_read = $3, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2
		  AND is_sender = false AND is_deleted = false AND is_draft = false
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID, read)
	if err != nil {
		return ErrQueryFail
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrMailNotFound
	}
	return nil
}
