package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

func (r *Repository) GetStarredEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMeta, error) {
}
