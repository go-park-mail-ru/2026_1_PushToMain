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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				return 0, ErrDuplicate
			}
			if pgErr.Code == ForeignKeyViolation {
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == UniqueViolation {
				return 0, ErrDuplicate
			}
			if pgErr.Code == ForeignKeyViolation {
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
	args := make([]any, 0, len(receiverIDs)*2)

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
    	WITH paginated_emails AS (
            SELECT
                e.id,
                e.sender_id,
                e.header,
                e.body,
                e.created_at,
                ue.is_read,
                ue.created_at AS received_at
            FROM user_emails ue
            JOIN emails e ON ue.email_id = e.id
            WHERE ue.receiver_id = $1
            ORDER BY ue.created_at DESC
            LIMIT $2 OFFSET $3
        ),
        receivers AS (
            SELECT
                ue.email_id,
                array_agg(u.email ORDER BY u.id) AS receivers_emails
            FROM user_emails ue
            JOIN users u ON ue.receiver_id = u.id
            WHERE ue.email_id IN (SELECT id FROM paginated_emails)
            GROUP BY ue.email_id
        )
        SELECT
            pe.*,
            COALESCE(r.receivers_emails, '{}'::text[]) AS receivers_emails
        FROM paginated_emails pe
        LEFT JOIN receivers r ON pe.id = r.email_id
        ORDER BY pe.received_at DESC;
    `

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]models.EmailWithMetadata, 0)
	for rows.Next() {
		var email models.EmailWithMetadata

		var receiversStr string

		err := rows.Scan(
			&email.ID,
			&email.SenderID,
			&email.Header,
			&email.Body,
			&email.CreatedAt,
			&email.IsRead,
			&email.ReceivedAt,
			&receiversStr,
		)
		if err != nil {
			return nil, err
		}

		receiversStr = strings.Trim(receiversStr, "{}")
		var receivers []string
		if receiversStr != "" {
			receivers = strings.Split(receiversStr, ",")
		}

		email.ReceiversEmails = receivers

		emails = append(emails, email)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

func (r *Repository) GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
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
        false as is_read,
        COALESCE(
            (SELECT json_agg(u.email)
             FROM user_emails ue
             JOIN users u ON ue.receiver_id = u.id
             WHERE ue.email_id = e.id),
            '[]'::json
        ) as receivers_emails
    FROM emails e
    WHERE e.sender_id = $1 AND is_deleted = false
    ORDER BY e.created_at DESC
    LIMIT $2 OFFSET $3
`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]models.EmailWithMetadata, 0)
	for rows.Next() {
		var email models.EmailWithMetadata
		var receiversEmailsJSON []byte

		err := rows.Scan(
			&email.ID,
			&email.SenderID,
			&email.Header,
			&email.Body,
			&email.CreatedAt,
			&email.IsRead,
			&receiversEmailsJSON,
		)
		if err != nil {
			return nil, ErrQueryFail
		}

		if err := json.Unmarshal(receiversEmailsJSON, &email.ReceiversEmails); err != nil {
			email.ReceiversEmails = []string{}
		}

		emails = append(emails, email)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

func (r *Repository) GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error) {
	query := `
        WITH email_data AS (
            SELECT
                e.id,
                e.sender_id,
                e.header,
                e.body,
                e.created_at,
                COALESCE(u.image_path, '') as image_path
            FROM emails e
            JOIN users u ON e.sender_id = u.id
            WHERE e.id = $1
        ),
        receivers AS (
            SELECT
                ue.email_id,
                array_agg(u.email ORDER BY u.id) AS receivers_emails
            FROM user_emails ue
            JOIN users u ON ue.receiver_id = u.id
            WHERE ue.email_id = $1
            GROUP BY ue.email_id
        )
        SELECT
            ed.id,
            ed.sender_id,
            ed.header,
            ed.body,
            ed.created_at,
            ed.image_path,
            COALESCE(r.receivers_emails, '{}'::text[]) AS receivers_emails
        FROM email_data ed
        LEFT JOIN receivers r ON ed.id = r.email_id
    `

	var email models.EmailWithAvatar
	var receiversStr string

	err := r.db.QueryRowContext(ctx, query, emailID).Scan(
		&email.ID,
		&email.SenderID,
		&email.Header,
		&email.Body,
		&email.CreatedAt,
		&email.SenderImagePath,
		&receiversStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMailNotFound
		}
		return nil, err
	}

	// Так же как в вашем примере
	receiversStr = strings.Trim(receiversStr, "{}")
	if receiversStr != "" {
		email.ReceiversEmails = strings.Split(receiversStr, ",")
	} else {
		email.ReceiversEmails = []string{}
	}

	return &email, nil
}

func (r *Repository) GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_emails
		WHERE receiver_id = $1 AND is_read = false
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, checkError(err)
	}

	return count, nil
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

func (r *Repository) GetUserEmailsCount(ctx context.Context, userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM emails
		WHERE sender_id = $1 and is_deleted = false
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

func (r *Repository) MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error {
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
		SET is_read = false, updated_at = NOW()
		WHERE email_id = $1 AND receiver_id = $2 AND is_read = true
	`

	_, err = r.db.ExecContext(ctx, query, emailID, userID)
	if err != nil {
		return checkError(err)
	}

	return nil
}

func (r *Repository) CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM user_emails
            WHERE email_id = $1 AND receiver_id = $2
        )
    `

	var exists bool
	err := r.db.QueryRowContext(ctx, query, emailID, userID).Scan(&exists)
	if err != nil {
		return false, ErrQueryFail
	}

	return exists, nil
}

func (r *Repository) DeleteEmailForReceiver(ctx context.Context, emailID, userID int64) error {
	query := `
        DELETE FROM user_emails
        WHERE email_id = $1 AND receiver_id = $2
    `

	result, err := r.db.ExecContext(ctx, query, emailID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrMailNotFound
	}

	return nil
}

func (r *Repository) DeleteEmailForSender(ctx context.Context, emailID, userID int64) error {
	query := `
        UPDATE emails
		SET is_deleted = true
		WHERE id = $1 AND sender_id = $2 AND is_deleted = false
    `

	result, err := r.db.ExecContext(ctx, query, emailID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrMailNotFound
	}

	return nil
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) error {
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
		return ErrAccessDenied
	}
	if !hasAccess {
		return ErrAccessDenied
	}

	return nil
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
