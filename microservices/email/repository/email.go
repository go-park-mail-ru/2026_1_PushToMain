package repository

import (
	"context"
	"database/sql"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) InsertEmail(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error) {
}

func (r *Repository) GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error) {
}

func (r *Repository) CheckEmailAccess(ctx context.Context, emailID, userID int64) error {}

func (r *Repository) GetInboxEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
}

func (r *Repository) GetSentEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error) {
}

func (r *Repository) GetSenderIDsByEmailIDs(ctx context.Context, emailIDs []int64) ([]int64, error) {}
