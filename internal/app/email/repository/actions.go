package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

func idsPlaceholders(ids []int64, startFrom int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	parts := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("$%d", startFrom+i)
		args[i] = id
	}
	return strings.Join(parts, ","), args
}

func (r *Repository) SetStarredBatch(ctx context.Context, userID int64, emailIDs []int64, starred bool) error {
	if len(emailIDs) == 0 {
		return nil
	}
	holders, idArgs := idsPlaceholders(emailIDs, 3)
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_starred = $2, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (%s)
	`, holders)

	args := append([]interface{}{userID, starred}, idArgs...)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) SetTrashedBatch(ctx context.Context, userID int64, emailIDs []int64, trashed bool) error {
	if len(emailIDs) == 0 {
		return nil
	}
	holders, idArgs := idsPlaceholders(emailIDs, 3)
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_deleted = $2, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (%s)
	`, holders)

	args := append([]interface{}{userID, trashed}, idArgs...)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) SetSpamBatch(ctx context.Context, userID int64, emailIDs []int64, spam bool) error {
	if len(emailIDs) == 0 {
		return nil
	}
	holders, idArgs := idsPlaceholders(emailIDs, 3)
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = $2, updated_at = NOW()
		WHERE user_id = $1 AND is_sender = false AND email_id IN (%s)
	`, holders)

	args := append([]interface{}{userID, spam}, idArgs...)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) MarkSendersAsSpamBatch(ctx context.Context, userID int64, emailIDs []int64) error {
	if len(emailIDs) == 0 {
		return nil
	}

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

	senderIDs, err := senderIDsForReceivedEmails(ctx, tx, userID, emailIDs)
	if err != nil {
		return err
	}
	if len(senderIDs) == 0 {
		// Нет писем, доступных текущему юзеру как получателю — нечего помечать.
		// Не считаем ошибкой — клиент мог прислать id уже из спама.
		if err = tx.Commit(); err != nil {
			return ErrTransactionFailed
		}
		committed = true
		return nil
	}

	if err = insertSpamSenders(ctx, tx, userID, senderIDs); err != nil {
		return err
	}

	if err = markEmailsFromSendersAsSpam(ctx, tx, userID, senderIDs); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}

func senderIDsForReceivedEmails(ctx context.Context, tx *sql.Tx, userID int64, emailIDs []int64) ([]int64, error) {
	holders, idArgs := idsPlaceholders(emailIDs, 2)
	query := fmt.Sprintf(`
		SELECT DISTINCT e.sender_id
		FROM emails e
		JOIN user_emails ue ON ue.email_id = e.id
		WHERE ue.user_id = $1 AND ue.is_sender = false
		  AND e.id IN (%s)
	`, holders)

	args := append([]interface{}{userID}, idArgs...)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrQueryFail
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return ids, nil
}

func insertSpamSenders(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	parts := make([]string, len(senderIDs))
	args := make([]interface{}, 0, len(senderIDs)+1)
	args = append(args, userID)
	for i, sid := range senderIDs {
		parts[i] = fmt.Sprintf("($1, $%d)", i+2)
		args = append(args, sid)
	}
	query := fmt.Sprintf(`
		INSERT INTO spam_senders (user_id, sender_id)
		VALUES %s
		ON CONFLICT (user_id, sender_id) DO NOTHING
	`, strings.Join(parts, ","))

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func markEmailsFromSendersAsSpam(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	holders, idArgs := idsPlaceholders(senderIDs, 2)
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = true, updated_at = NOW()
		WHERE user_id = $1 AND is_sender = false AND is_spam = false
		  AND email_id IN (
		      SELECT id FROM emails WHERE sender_id IN (%s)
		  )
	`, holders)

	args := append([]interface{}{userID}, idArgs...)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) HardDeleteBatch(ctx context.Context, userID int64, emailIDs []int64) error {
	if len(emailIDs) == 0 {
		return nil
	}
	holders, idArgs := idsPlaceholders(emailIDs, 2)
	query := fmt.Sprintf(`
		DELETE FROM user_emails
		WHERE user_id = $1 AND email_id IN (%s)
	`, holders)

	args := append([]interface{}{userID}, idArgs...)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UnmarkSendersAsSpamBatch(ctx context.Context, userID int64, emailIDs []int64) error {
	if len(emailIDs) == 0 {
		return nil
	}

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

	senderIDs, err := senderIDsForEmails(ctx, tx, emailIDs)
	if err != nil {
		return err
	}
	if len(senderIDs) == 0 {
		if err = tx.Commit(); err != nil {
			return ErrTransactionFailed
		}
		committed = true
		return nil
	}

	if err = deleteSpamSenders(ctx, tx, userID, senderIDs); err != nil {
		return err
	}

	if err = unmarkEmailsFromSendersAsSpam(ctx, tx, userID, senderIDs); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}

func senderIDsForEmails(ctx context.Context, tx *sql.Tx, emailIDs []int64) ([]int64, error) {
	holders, idArgs := idsPlaceholders(emailIDs, 1)
	query := fmt.Sprintf(`
		SELECT DISTINCT sender_id
		FROM emails
		WHERE id IN (%s)
	`, holders)

	rows, err := tx.QueryContext(ctx, query, idArgs...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrQueryFail
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return ids, nil
}

func deleteSpamSenders(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	holders, idArgs := idsPlaceholders(senderIDs, 2)
	query := fmt.Sprintf(`
		DELETE FROM spam_senders
		WHERE user_id = $1 AND sender_id IN (%s)
	`, holders)

	args := append([]interface{}{userID}, idArgs...)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func unmarkEmailsFromSendersAsSpam(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	holders, idArgs := idsPlaceholders(senderIDs, 2)
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = false, updated_at = NOW()
		WHERE user_id = $1
			AND is_sender = false
			AND is_spam = true
		  	AND email_id IN (SELECT id FROM emails WHERE sender_id IN (%s))
	`, holders)

	args := append([]interface{}{userID}, idArgs...)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) GetUserEmailFlags(ctx context.Context, emailID, userID int64, isSender bool) (*models.UserEmail, error) {
	const query = `
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
