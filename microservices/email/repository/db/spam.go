package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (r *Repository) InsertSpamSender(ctx context.Context, tx *sql.Tx, userID int64, senderID int64) error {
	const query = `
		INSERT INTO spam_senders (user_id, sender_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, sender_id) DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, query, userID, senderID); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) GetSenderIDsByEmailIDs(ctx context.Context, userID int64, emailIDs []int64) ([]int64, error) {
	if len(emailIDs) == 0 {
		return []int64{}, nil
	}
	parts := make([]string, len(emailIDs))
	args := make([]any, 0, len(emailIDs)+1)
	args = append(args, userID)
	for i, id := range emailIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		SELECT DISTINCT e.sender_id
		FROM emails e
		JOIN user_emails ue ON ue.email_id = e.id AND ue.user_id = $1
		WHERE e.id IN (%s) AND e.sender_id IS NOT NULL
	`, strings.Join(parts, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrQueryFail
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return out, nil
}

func (r *Repository) DeleteSpamSendersBatchTx(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	if len(senderIDs) == 0 {
		return nil
	}
	parts := make([]string, len(senderIDs))
	args := make([]any, 0, len(senderIDs)+1)
	args = append(args, userID)
	for i, id := range senderIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		DELETE FROM spam_senders
		WHERE user_id = $1 AND sender_id IN (%s)
	`, strings.Join(parts, ","))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UnmarkSpamForSendersTx(ctx context.Context, tx *sql.Tx, userID int64, senderIDs []int64) error {
	if len(senderIDs) == 0 {
		return nil
	}
	parts := make([]string, len(senderIDs))
	args := make([]any, 0, len(senderIDs)+1)
	args = append(args, userID)
	for i, id := range senderIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = false, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (
			SELECT id FROM emails WHERE sender_id IN (%s)
		)
	`, strings.Join(parts, ","))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) SpamEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	if len(emailIDs) == 0 {
		return nil
	}
	parts := make([]string, len(emailIDs))
	args := make([]any, 0, len(emailIDs)+1)
	args = append(args, userID)
	for i, id := range emailIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = true, is_starred = false, is_deleted = false, updated_at = NOW()
		WHERE user_id = $1
		  AND email_id IN (%s)
		  AND email_id NOT IN (SELECT id FROM emails WHERE sender_id = $1)
	`, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UnspamEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	return r.setUserEmailFlagsBatch(ctx, userID, emailIDs, "is_spam", false)
}

func (r *Repository) GetSpamEmailIDs(ctx context.Context, userID int64, emailIDs []int64) ([]int64, error) {
	if len(emailIDs) == 0 {
		return nil, nil
	}
	parts := make([]string, len(emailIDs))
	args := make([]any, 0, len(emailIDs)+1)
	args = append(args, userID)
	for i, id := range emailIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		SELECT email_id FROM user_emails
		WHERE user_id = $1 AND is_spam = true AND email_id IN (%s)
	`, strings.Join(parts, ","))
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer func() {
		_ = rows.Close()
	}()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, ErrQueryFail
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrQueryFail
	}
	return out, nil
}

func (r *Repository) BlockSendersBatch(ctx context.Context, userID int64, senderIDs []int64) error {
	if len(senderIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrQueryFail
	}
	defer func() {
		_ = tx.Rollback()
	}()

	insertParts := make([]string, len(senderIDs))
	insertArgs := make([]any, 0, len(senderIDs)+1)
	insertArgs = append(insertArgs, userID)
	for i, id := range senderIDs {
		insertParts[i] = fmt.Sprintf("($1, $%d)", i+2)
		insertArgs = append(insertArgs, id)
	}
	insertQuery := fmt.Sprintf(`
		INSERT INTO spam_senders (user_id, sender_id)
		VALUES %s
		ON CONFLICT (user_id, sender_id) DO NOTHING
	`, strings.Join(insertParts, ","))
	if _, err := tx.ExecContext(ctx, insertQuery, insertArgs...); err != nil {
		return ErrQueryFail
	}

	updateParts := make([]string, len(senderIDs))
	updateArgs := make([]any, 0, len(senderIDs)+1)
	updateArgs = append(updateArgs, userID)
	for i, id := range senderIDs {
		updateParts[i] = fmt.Sprintf("$%d", i+2)
		updateArgs = append(updateArgs, id)
	}
	updateQuery := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = true, is_starred = false, is_deleted = false, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (
			SELECT id FROM emails WHERE sender_id IN (%s)
		)
	`, strings.Join(updateParts, ","))
	if _, err := tx.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
		return ErrQueryFail
	}

	if err := tx.Commit(); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UnblockSendersBatch(ctx context.Context, userID int64, senderIDs []int64) error {
	if len(senderIDs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrQueryFail
	}
	defer func() {
		_ = tx.Rollback()
	}()

	parts := make([]string, len(senderIDs))
	args := make([]any, 0, len(senderIDs)+1)
	args = append(args, userID)
	for i, id := range senderIDs {
		parts[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	deleteQuery := fmt.Sprintf(`
		DELETE FROM spam_senders
		WHERE user_id = $1 AND sender_id IN (%s)
	`, strings.Join(parts, ","))
	if _, err := tx.ExecContext(ctx, deleteQuery, args...); err != nil {
		return ErrQueryFail
	}

	updateQuery := fmt.Sprintf(`
		UPDATE user_emails
		SET is_spam = false, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (
			SELECT id FROM emails WHERE sender_id IN (%s)
		)
	`, strings.Join(parts, ","))
	if _, err := tx.ExecContext(ctx, updateQuery, args...); err != nil {
		return ErrQueryFail
	}

	if err := tx.Commit(); err != nil {
		return ErrQueryFail
	}
	return nil
}
