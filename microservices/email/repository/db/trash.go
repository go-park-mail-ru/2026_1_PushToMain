package repository

import (
	"context"
	"fmt"
	"strings"
)

func (r *Repository) DeleteUserEmailsBatch(ctx context.Context, userID int64, emailIDs []int64) error {
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
		DELETE FROM user_emails
		WHERE user_id = $1 AND email_id IN (%s)
	`, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) TrashEmails(ctx context.Context, userID int64, emailIDs []int64) error {
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
		SET is_deleted = true, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (%s) AND is_spam = false
	`, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) UntrashEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	return r.setUserEmailFlagsBatch(ctx, userID, emailIDs, "is_deleted", false)
}
