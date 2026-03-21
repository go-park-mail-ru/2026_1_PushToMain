package service

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

type Repository interface {
	GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error) {
	return s.repo.GetEmailsByReceiver(ctx, userID)
}
