package repository

import (
	"context"
	"fmt"
	"strings"
)

func (r *Repository) setUserEmailFlagsBatch(
	ctx context.Context,
	userID int64,
	emailIDs []int64,
	column string,
	value bool,
) error {
	if len(emailIDs) == 0 {
		return nil
	}
	parts := make([]string, len(emailIDs))
	args := make([]any, 0, len(emailIDs)+2)
	args = append(args, userID, value)
	for i, id := range emailIDs {
		parts[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE user_emails
		SET %s = $2, updated_at = NOW()
		WHERE user_id = $1 AND email_id IN (%s)
	`, column, strings.Join(parts, ","))
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return ErrQueryFail
	}
	return nil
}

func (r *Repository) ReadEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	return r.setUserEmailFlagsBatch(ctx, userID, emailIDs, "is_read", true)
}

func (r *Repository) UnreadEmails(ctx context.Context, userID int64, emailIDs []int64) error {
	return r.setUserEmailFlagsBatch(ctx, userID, emailIDs, "is_read", false)
}
