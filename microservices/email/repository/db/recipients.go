package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) InsertEmailRecipients(ctx context.Context, tx *sql.Tx, emailID int64, recipients []models.Recipient) error {
	if len(recipients) == 0 {
		return nil
	}

	parts := make([]string, len(recipients))
	args := make([]any, 0, len(recipients)*2+1)
	args = append(args, emailID)
	for i, rec := range recipients {
		parts[i] = fmt.Sprintf("($1, $%d, $%d)", i*2+2, i*2+3)
		args = append(args, rec.UserID, rec.Email)
	}

	query := fmt.Sprintf(`
        INSERT INTO email_recipients (email_id, recipient_user_id, recipient_email)
        VALUES %s
        ON CONFLICT (email_id, recipient_email) DO UPDATE
            SET recipient_user_id = COALESCE(email_recipients.recipient_user_id, EXCLUDED.recipient_user_id)
    `, strings.Join(parts, ","))

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return mapPgErrorReceiver(err)
	}
	return nil
}

func (r *Repository) InsertUserEmail(ctx context.Context, tx *sql.Tx, userID, emailID int64, isSender, checkSpam bool) error {
	const query = `
        INSERT INTO user_emails (user_id, email_id, is_sender, is_spam)
        SELECT $1, $2, $3, $4 AND EXISTS (
            SELECT 1
            FROM spam_senders ss
            JOIN emails e ON e.id = $2
            WHERE ss.user_id = $1 AND ss.sender_id = e.sender_id
        )
       
    `

	if _, err := tx.ExecContext(ctx, query, userID, emailID, isSender, checkSpam); err != nil {
		return mapPgError(err)
	}
	return nil
}

func insertEmailRecipients(ctx context.Context, tx *sql.Tx, emailID int64, recipients []string) error {
	if len(recipients) == 0 {
		return nil
	}
	parts := make([]string, len(recipients))
	args := make([]any, 0, len(recipients)+1)
	args = append(args, emailID)
	for i, r := range recipients {
		parts[i] = fmt.Sprintf("($1, $%d)", i+2)
		args = append(args, r)
	}
	query := fmt.Sprintf(`
		INSERT INTO email_recipients (email_id, recipient_email)
		VALUES %s
		ON CONFLICT (email_id, recipient_email) DO NOTHING
	`, strings.Join(parts, ","))

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}
