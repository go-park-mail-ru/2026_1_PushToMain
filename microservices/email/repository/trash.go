package repository

import (
	"context"
)

func (r *Repository) GetTrashEmailIDs(ctx context.Context, userID int64, limit, offset int) ([]int64, error) {
	const query = `
		SELECT email_id
		FROM user_emails
		WHERE user_id = $1 AND is_deleted = true
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryFail
	}
	defer rows.Close()

	ids := make([]int64, 0)
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
