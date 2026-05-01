package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/jackc/pgx/v5/pgconn"
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
	ErrAccessDenied      = errors.New("have no access")
	ErrDraftNotFound     = errors.New("draft not found")
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
	err := r.db.QueryRowContext(ctx, query, email.SenderID, email.Header, email.Body).Scan(&emailID)
	if err != nil {
		return 0, mapPgError(err)
	}
	return emailID, nil
}

func (r *Repository) SaveEmailWithTx(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error) {
	query := `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var emailID int64
	err := tx.QueryRowContext(ctx, query, email.SenderID, email.Header, email.Body).Scan(&emailID)
	if err != nil {
		return 0, mapPgError(err)
	}
	return emailID, nil
}

func (r *Repository) AddEmailUserWithTx(ctx context.Context, tx *sql.Tx, emailID int64, userID int64, isSender bool) error {
	query := `
		INSERT INTO user_emails (user_id, email_id, is_sender, is_spam)
		SELECT $1, $2, $3,
			CASE WHEN $3 THEN false
			ELSE EXISTS(
				SELECT 1
				FROM spam_senders ss
				JOIN emails e ON e.id = $2
				WHERE ss.user_id = $1 AND ss.sender_id = e.sender_id
			)
			END
	`
	_, err := tx.ExecContext(ctx, query, userID, emailID, isSender)
	if err != nil {
		return mapPgErrorReceiver(err)
	}
	return nil
}

func (r *Repository) GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)

	query := `
		WITH paginated_emails AS (
			SELECT
				e.id, e.sender_id, e.header, e.body, e.created_at,
				ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
				ue.created_at AS received_at
			FROM user_emails ue
			JOIN emails e ON ue.email_id = e.id
			WHERE ue.user_id = $1
			  AND ue.is_sender = false
			  AND ue.is_deleted = false
			  AND ue.is_spam = false
			  AND ue.is_draft = false
			ORDER BY ue.created_at DESC
			LIMIT $2 OFFSET $3
		),
		receivers AS (
			SELECT ue.email_id, array_agg(u.email ORDER BY u.id) AS receivers_emails
			FROM user_emails ue
			JOIN users u ON ue.user_id = u.id
			WHERE ue.email_id IN (SELECT id FROM paginated_emails) AND ue.is_sender = false
			GROUP BY ue.email_id
		)
		SELECT pe.*, COALESCE(r.receivers_emails, '{}'::text[])
		FROM paginated_emails pe
		LEFT JOIN receivers r ON pe.id = r.email_id
		ORDER BY pe.received_at DESC
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)

	query := `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at,
			false AS is_read, ue.is_starred, false AS is_spam, ue.is_deleted,
			COALESCE(
				(SELECT json_agg(u.email)
				 FROM user_emails ue2
				 JOIN users u ON ue2.user_id = u.id
				 WHERE ue2.email_id = e.id AND ue2.is_sender = false),
				'[]'::json
			) AS receivers_emails
		FROM emails e
		JOIN user_emails ue ON e.id = ue.email_id
		WHERE e.sender_id = $1
		  AND ue.user_id = $1
		  AND ue.is_sender = true
		  AND ue.is_deleted = false
		  AND ue.is_draft = false
		ORDER BY e.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	emails := make([]models.EmailWithMetadata, 0)
	for rows.Next() {
		var em models.EmailWithMetadata
		var receiversJSON []byte
		if err := rows.Scan(
			&em.ID, &em.SenderID, &em.Header, &em.Body, &em.CreatedAt,
			&em.IsRead, &em.IsStarred, &em.IsSpam, &em.IsDeleted, &receiversJSON,
		); err != nil {
			return nil, ErrQueryFail
		}
		if err := json.Unmarshal(receiversJSON, &em.ReceiversEmails); err != nil {
			em.ReceiversEmails = []string{}
		}
		emails = append(emails, em)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return emails, nil
}

func (r *Repository) GetSpamEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)

	query := `
		WITH paginated_emails AS (
			SELECT e.id, e.sender_id, e.header, e.body, e.created_at,
				ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
				ue.created_at AS received_at
			FROM user_emails ue
			JOIN emails e ON ue.email_id = e.id
			WHERE ue.user_id = $1
			  AND ue.is_sender = false
			  AND ue.is_spam = true
			  AND ue.is_deleted = false
			  AND ue.is_draft = false
			ORDER BY ue.created_at DESC
			LIMIT $2 OFFSET $3
		),
		receivers AS (
			SELECT ue.email_id, array_agg(u.email ORDER BY u.id) AS receivers_emails
			FROM user_emails ue
			JOIN users u ON ue.user_id = u.id
			WHERE ue.email_id IN (SELECT id FROM paginated_emails) AND ue.is_sender = false
			GROUP BY ue.email_id
		)
		SELECT pe.*, COALESCE(r.receivers_emails, '{}'::text[])
		FROM paginated_emails pe
		LEFT JOIN receivers r ON pe.id = r.email_id
		ORDER BY pe.received_at DESC
	`
	return r.queryEmailsList(ctx, query, userID, limit, offset)
}

func (r *Repository) GetTrashEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
	limit, offset = normPage(limit, offset)

	query := `
		WITH paginated_emails AS (
			SELECT e.id, e.sender_id, e.header, e.body, e.created_at,
				ue.is_read, ue.is_starred, ue.is_spam, ue.is_deleted,
				ue.updated_at AS received_at
			FROM user_emails ue
			JOIN emails e ON ue.email_id = e.id
			WHERE ue.user_id = $1
			  AND ue.is_deleted = true
			  AND ue.is_draft = false
			ORDER BY ue.updated_at DESC
			LIMIT $2 OFFSET $3
		),
		receivers AS (
			SELECT ue.email_id, array_agg(u.email ORDER BY u.id) AS receivers_emails
			FROM user_emails ue
			JOIN users u ON ue.user_id = u.id
			WHERE ue.email_id IN (SELECT id FROM paginated_emails) AND ue.is_sender = false
			GROUP BY ue.email_id
		)
		SELECT pe.*, COALESCE(r.receivers_emails, '{}'::text[])
		FROM paginated_emails pe
		LEFT JOIN receivers r ON pe.id = r.email_id
		ORDER BY pe.received_at DESC
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

func (r *Repository) GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error) {
	query := `
		WITH email_data AS (
			SELECT e.id, e.sender_id, e.header, e.body, e.created_at,
				COALESCE(u.image_path, '') AS image_path
			FROM emails e
			JOIN users u ON e.sender_id = u.id
			WHERE e.id = $1
		),
		receivers AS (
			SELECT ue.email_id, array_agg(u.email ORDER BY u.id) AS receivers_emails
			FROM user_emails ue
			JOIN users u ON ue.user_id = u.id
			WHERE ue.email_id = $1 AND ue.is_sender = false
			GROUP BY ue.email_id
		)
		SELECT ed.id, ed.sender_id, ed.header, ed.body, ed.created_at,
			ed.image_path,
			COALESCE(r.receivers_emails, '{}'::text[])
		FROM email_data ed
		LEFT JOIN receivers r ON ed.id = r.email_id
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

func (r *Repository) GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_read = false AND is_sender = false
		  AND is_deleted = false AND is_spam = false AND is_draft = false
	`, userID)
}

func (r *Repository) GetEmailsCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_deleted = false AND is_spam = false AND is_draft = false
	`, userID)
}

func (r *Repository) GetSenderEmailsCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = true
		  AND is_deleted = false AND is_draft = false
	`, userID)
}

func (r *Repository) GetSpamEmailsCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_spam = true AND is_deleted = false AND is_draft = false
	`, userID)
}

func (r *Repository) GetUnreadSpamCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = false
		  AND is_spam = true AND is_read = false
		  AND is_deleted = false AND is_draft = false
	`, userID)
}

func (r *Repository) GetTrashEmailsCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_deleted = true AND is_draft = false
	`, userID)
}

func (r *Repository) GetUnreadTrashCount(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_deleted = true AND is_read = false
		  AND is_sender = false AND is_draft = false
	`, userID)
}

func (r *Repository) scanCount(ctx context.Context, query string, args ...interface{}) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, ErrQueryFail
	}
	return count, nil
}

func (r *Repository) MarkEmailAsRead(ctx context.Context, emailID, userID int64) error {
	return r.toggleReadFlag(ctx, emailID, userID, true)
}

func (r *Repository) MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error {
	return r.toggleReadFlag(ctx, emailID, userID, false)
}

func (r *Repository) toggleReadFlag(ctx context.Context, emailID, userID int64, read bool) error {
	query := `
		UPDATE user_emails
		SET is_read = $3, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_sender = false AND is_deleted = false AND is_draft = false
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

func (r *Repository) CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_emails WHERE email_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, emailID, userID).Scan(&exists)
	if err != nil {
		return false, ErrQueryFail
	}
	return exists, nil
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) error {
	exists, err := r.CheckUserEmailExists(ctx, emailID, userID)
	if err != nil {
		return ErrAccessDenied
	}
	if !exists {
		return ErrAccessDenied
	}
	return nil
}

func (r *Repository) GetUserEmailFlags(ctx context.Context, emailID, userID int64, isSender bool) (*models.UserEmail, error) {
	query := `
		SELECT id, email_id, user_id, is_sender, is_read, is_deleted, is_starred, is_spam, is_draft
		FROM user_emails
		WHERE email_id = $1 AND user_id = $2 AND is_sender = $3
	`
	var ue models.UserEmail
	err := r.db.QueryRowContext(ctx, query, emailID, userID, isSender).Scan(
		&ue.ID, &ue.EmailID, &ue.UserID, &ue.IsSender,
		&ue.IsRead, &ue.IsDeleted, &ue.IsStarred, &ue.IsSpam, &ue.IsDraft,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMailNotFound
		}
		return nil, ErrQueryFail
	}
	return &ue, nil
}

func (r *Repository) SoftDeleteUserEmail(ctx context.Context, emailID, userID int64, isSender bool) error {
	query := `
		UPDATE user_emails
		SET is_deleted = true, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_sender = $3 AND is_deleted = false
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID, isSender)
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

func (r *Repository) HardDeleteUserEmail(ctx context.Context, emailID, userID int64, isSender bool) error {
	query := `
		DELETE FROM user_emails
		WHERE email_id = $1 AND user_id = $2 AND is_sender = $3
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID, isSender)
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

func (r *Repository) RestoreFromTrash(ctx context.Context, emailID, userID int64) error {
	query := `
		UPDATE user_emails
		SET is_deleted = false, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_deleted = true
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID)
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

func (r *Repository) SetStarred(ctx context.Context, emailID, userID int64, starred bool) error {
	query := `
		UPDATE user_emails
		SET is_starred = $3, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID, starred)
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

func (r *Repository) MarkSenderAsSpam(ctx context.Context, emailID, userID int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, ErrTransactionFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var senderID int64
	err = tx.QueryRowContext(ctx, `
		SELECT e.sender_id
		FROM emails e
		JOIN user_emails ue ON ue.email_id = e.id
		WHERE e.id = $1 AND ue.user_id = $2 AND ue.is_sender = false
		LIMIT 1
	`, emailID, userID).Scan(&senderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrMailNotFound
		}
		return 0, ErrQueryFail
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO spam_senders (user_id, sender_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, sender_id) DO NOTHING
	`, userID, senderID); err != nil {
		return 0, ErrQueryFail
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE user_emails
		SET is_spam = true, updated_at = NOW()
		WHERE user_id = $1
		  AND is_sender = false
		  AND is_spam = false
		  AND email_id IN (SELECT id FROM emails WHERE sender_id = $2)
	`, userID, senderID)
	if err != nil {
		return 0, ErrQueryFail
	}
	affected, _ := res.RowsAffected()

	if err = tx.Commit(); err != nil {
		return 0, ErrTransactionFailed
	}
	committed = true
	return affected, nil
}

func (r *Repository) MoveToTrash(ctx context.Context, emailID, userID int64) error {
	query := `
		UPDATE user_emails
		SET is_deleted = true, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_deleted = false
	`
	res, err := r.db.ExecContext(ctx, query, emailID, userID)
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

func (r *Repository) CountDraftsByUser(ctx context.Context, userID int64) (int, error) {
	return r.scanCount(ctx, `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = true AND is_draft = true
	`, userID)
}

func (r *Repository) CreateDraft(ctx context.Context, draft models.Draft) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, ErrTransactionFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var emailID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`, draft.SenderID, draft.Header, draft.Body).Scan(&emailID); err != nil {
		return 0, ErrSaveEmail
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO user_emails (user_id, email_id, is_sender, is_draft)
		VALUES ($1, $2, true, true)
	`, draft.SenderID, emailID); err != nil {
		return 0, ErrSaveEmail
	}

	if err = insertDraftReceivers(ctx, tx, emailID, draft.Receivers); err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, ErrTransactionFailed
	}
	committed = true
	return emailID, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, draft models.Draft) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrTransactionFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `
		UPDATE emails e
		SET header = $1, body = $2
		FROM user_emails ue
		WHERE e.id = $3
		  AND ue.email_id = e.id
		  AND ue.user_id = $4
		  AND ue.is_sender = true
		  AND ue.is_draft = true
	`, draft.Header, draft.Body, draft.ID, draft.SenderID)
	if err != nil {
		return ErrQueryFail
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrDraftNotFound
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE user_emails
		SET updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_sender = true AND is_draft = true
	`, draft.ID, draft.SenderID); err != nil {
		return ErrQueryFail
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM draft_receivers WHERE email_id = $1`, draft.ID); err != nil {
		return ErrQueryFail
	}
	if err = insertDraftReceivers(ctx, tx, draft.ID, draft.Receivers); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}

func (r *Repository) GetDraftByID(ctx context.Context, draftID, userID int64) (*models.Draft, error) {
	query := `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at,
			COALESCE(
				(SELECT array_agg(dr.receiver_email ORDER BY dr.id)
				 FROM draft_receivers dr
				 WHERE dr.email_id = e.id),
				'{}'::text[]
			)
		FROM emails e
		JOIN user_emails ue ON ue.email_id = e.id
		WHERE e.id = $1
		  AND ue.user_id = $2
		  AND ue.is_sender = true
		  AND ue.is_draft = true
	`
	var d models.Draft
	var receiversStr string
	err := r.db.QueryRowContext(ctx, query, draftID, userID).Scan(
		&d.ID, &d.SenderID, &d.Header, &d.Body, &d.CreatedAt, &d.UpdatedAt, &receiversStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDraftNotFound
		}
		return nil, ErrQueryFail
	}
	d.Receivers = parsePgTextArray(receiversStr)
	return &d, nil
}

func (r *Repository) GetDrafts(ctx context.Context, userID int64, limit, offset int) ([]models.Draft, error) {
	limit, offset = normPage(limit, offset)

	query := `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at,
			COALESCE(
				(SELECT array_agg(dr.receiver_email ORDER BY dr.id)
				 FROM draft_receivers dr
				 WHERE dr.email_id = e.id),
				'{}'::text[]
			)
		FROM emails e
		JOIN user_emails ue ON ue.email_id = e.id
		WHERE ue.user_id = $1 AND ue.is_sender = true AND ue.is_draft = true
		ORDER BY ue.updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	drafts := make([]models.Draft, 0)
	for rows.Next() {
		var d models.Draft
		var receiversStr string
		if err := rows.Scan(&d.ID, &d.SenderID, &d.Header, &d.Body, &d.CreatedAt, &d.UpdatedAt, &receiversStr); err != nil {
			return nil, ErrQueryFail
		}
		d.Receivers = parsePgTextArray(receiversStr)
		drafts = append(drafts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return drafts, nil
}

func (r *Repository) DeleteDraft(ctx context.Context, draftID, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrTransactionFailed
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `
		DELETE FROM user_emails
		WHERE email_id = $1 AND user_id = $2 AND is_sender = true AND is_draft = true
	`, draftID, userID)
	if err != nil {
		return ErrQueryFail
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrDraftNotFound
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM emails WHERE id = $1`, draftID); err != nil {
		return ErrQueryFail
	}

	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}

func (r *Repository) MarkDraftAsSentTx(ctx context.Context, tx *sql.Tx, draftID, userID int64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE user_emails
		SET is_draft = false, updated_at = NOW()
		WHERE email_id = $1 AND user_id = $2 AND is_sender = true AND is_draft = true
	`, draftID, userID)
	if err != nil {
		return ErrQueryFail
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ErrQueryFail
	}
	if rows == 0 {
		return ErrDraftNotFound
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM draft_receivers WHERE email_id = $1`, draftID); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func insertDraftReceivers(ctx context.Context, tx *sql.Tx, emailID int64, receivers []string) error {
	if len(receivers) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(receivers))
	args := make([]interface{}, 0, len(receivers)+1)
	args = append(args, emailID)
	for i, rcv := range receivers {
		placeholders = append(placeholders, fmt.Sprintf("($1, $%d)", i+2))
		args = append(args, rcv)
	}
	query := fmt.Sprintf(
		`INSERT INTO draft_receivers (email_id, receiver_email) VALUES %s`,
		strings.Join(placeholders, ", "),
	)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrReceiverAdd
	}
	return nil
}

func parsePgTextArray(s string) []string {
	s = strings.Trim(s, "{}")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func normPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == UniqueViolation {
			return ErrDuplicate
		}
		if pgErr.Code == ForeignKeyViolation {
			return ErrForeignKey
		}
	}
	return ErrSaveEmail
}

func mapPgErrorReceiver(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == UniqueViolation {
			return ErrDuplicate
		}
		if pgErr.Code == ForeignKeyViolation {
			return ErrForeignKey
		}
	}
	return ErrReceiverAdd
}
