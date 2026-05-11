package repository

import (
	"context"
	"database/sql"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) InsertEmailRecipients(ctx context.Context, tx *sql.Tx, emailID int64, recipients []models.Recipient) error {
}

func (r *Repository) InsertUserEmail(ctx context.Context, tx *sql.Tx, userID, emailID int64, checkSpam bool) error {
}

func (r *Repository) GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error) {}
