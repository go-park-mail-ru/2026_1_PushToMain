package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

func (r *Repository) CountDraftsByUser(ctx context.Context, userID int64) (int, error) {
	const query = `
		SELECT COUNT(*) FROM user_emails
		WHERE user_id = $1 AND is_sender = true AND is_draft = true
	`
	return r.scanCount(ctx, query, userID)
}

// CreateDraft — три INSERT'а в одной транзакции:
//  1. emails (тело черновика);
//  2. user_emails (связь автор-черновик с is_draft=true);
//  3. draft_receivers (адреса получателей как plain text).
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
	err = tx.QueryRowContext(ctx, `
		INSERT INTO emails (sender_id, header, body)
		VALUES ($1, $2, $3)
		RETURNING id
	`, draft.SenderID, draft.Header, draft.Body).Scan(&emailID)
	if err != nil {
		return 0, ErrSaveEmail
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_emails (user_id, email_id, is_sender, is_draft)
		VALUES ($1, $2, true, true)
	`, draft.SenderID, emailID)
	if err != nil {
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

// UpdateDraft — replace-семантика: переписываем header/body, удаляем старых получателей,
// вставляем новых. Всё в одной транзакции.
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
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at,
			COALESCE((
				SELECT array_agg(dr.receiver_email ORDER BY dr.id)
				FROM draft_receivers dr
				WHERE dr.email_id = e.id
			), '{}'::text[]) AS receivers
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
	const query = `
		SELECT
			e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at,
			COALESCE((
				SELECT array_agg(dr.receiver_email ORDER BY dr.id)
				FROM draft_receivers dr
				WHERE dr.email_id = e.id
			), '{}'::text[]) AS receivers
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

func (r *Repository) DeleteDraftsBatch(ctx context.Context, userID int64, draftIDs []int64) error {
	if len(draftIDs) == 0 {
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

	holders1, idArgs := idsPlaceholders(draftIDs, 2)
	args := append([]any{userID}, idArgs...)

	q1 := fmt.Sprintf(`
		DELETE FROM user_emails
		WHERE user_id = $1 AND is_sender = true AND is_draft = true
		  AND email_id IN (%s)
	`, holders1)
	if _, err = tx.ExecContext(ctx, q1, args...); err != nil {
		return ErrQueryFail
	}

	holders2, _ := idsPlaceholders(draftIDs, 1)
	q2 := fmt.Sprintf(`DELETE FROM emails WHERE id IN (%s)`, holders2)
	if _, err = tx.ExecContext(ctx, q2, idArgs...); err != nil {
		return ErrQueryFail
	}

	if err = tx.Commit(); err != nil {
		return ErrTransactionFailed
	}
	committed = true
	return nil
}

// MarkDraftAsSentTx — перевод черновика в "отправленное" состояние внутри транзакции отправки.
// Чистит draft_receivers (они нужны были только для черновика).
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

func insertDraftReceivers(ctx context.Context, tx *sql.Tx, emailID int64, receivers []string) error {
	if len(receivers) == 0 {
		return nil
	}
	parts := make([]string, len(receivers))
	args := make([]any, 0, len(receivers)+1)
	args = append(args, emailID)
	for i, rcv := range receivers {
		parts[i] = fmt.Sprintf("($1, $%d)", i+2)
		args = append(args, rcv)
	}
	query := fmt.Sprintf(
		`INSERT INTO draft_receivers (email_id, receiver_email) VALUES %s`,
		strings.Join(parts, ","),
	)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return ErrReceiverAdd
	}
	return nil
}
