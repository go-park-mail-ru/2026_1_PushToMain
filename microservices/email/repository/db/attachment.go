package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) InsertAttachment(
	ctx context.Context,
	tx *sql.Tx,
	attachment models.Attachment,
) (int64, error) {
	const query = `
		INSERT INTO attachments (
			email_id,
			filename,
			content_type,
			size_bytes,
			storage_path
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int64
	err := tx.QueryRowContext(
		ctx,
		query,
		attachment.EmailID,
		attachment.FileName,
		attachment.ContentType,
		attachment.SizeBytes,
		attachment.StoragePath,
	).Scan(&id)

	return id, err
}

func (r *Repository) GetAttachmentsByEmailIDs(
	ctx context.Context,
	emailIDs []int64,
) ([]models.Attachment, error) {
	if len(emailIDs) == 0 {
		return []models.Attachment{}, nil
	}

	placeholders, args := buildInQuery(emailIDs, 1)

	query := fmt.Sprintf(`
		SELECT
			id,
			email_id,
			filename,
			content_type,
			size_bytes,
			storage_path,
			created_at
		FROM attachments
		WHERE email_id IN (%s)
	`, placeholders)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var attachments []models.Attachment

	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(
			&a.ID,
			&a.EmailID,
			&a.FileName,
			&a.ContentType,
			&a.SizeBytes,
			&a.StoragePath,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return attachments, nil
}

func (r *Repository) DeleteAttachments(
	ctx context.Context,
	tx *sql.Tx,
	ids []int64,
) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders, args := buildInQuery(ids, 1)

	query := fmt.Sprintf(`
		DELETE FROM attachments
		WHERE id IN (%s)
	`, placeholders)

	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

// GetAttachmentByID returns a single attachment, checking it belongs to emailID.
func (r *Repository) GetAttachmentByID(
	ctx context.Context,
	attachmentID, emailID int64,
) (*models.Attachment, error) {
	const query = `
		SELECT id, email_id, filename, content_type, size_bytes, storage_path, created_at
		FROM attachments
		WHERE id = $1 AND email_id = $2
	`
	var a models.Attachment
	err := r.db.QueryRowContext(ctx, query, attachmentID, emailID).Scan(
		&a.ID, &a.EmailID, &a.FileName, &a.ContentType, &a.SizeBytes, &a.StoragePath, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) FillAttachments(
	ctx context.Context,
	emails []models.Email,
) error {
	if len(emails) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(emails))
	idxByID := make(map[int64]int)

	for i, email := range emails {
		ids = append(ids, email.ID)
		idxByID[email.ID] = i
	}

	attachments, err := r.GetAttachmentsByEmailIDs(ctx, ids)
	if err != nil {
		return err
	}

	for _, a := range attachments {
		idx := idxByID[a.EmailID]
		emails[idx].Attachments = append(emails[idx].Attachments, a)
	}

	return nil
}

func buildInQuery(ids []int64, start int) (string, []interface{}) {
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))

	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", start+i))
		args = append(args, id)
	}

	return strings.Join(placeholders, ","), args
}
