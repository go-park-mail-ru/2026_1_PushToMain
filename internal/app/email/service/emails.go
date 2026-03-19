package service

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

type EmailRepository interface {
	GetAll(ctx context.Context) ([]models.Email, error)
}

type EmailService struct {
	repo EmailRepository
}

func NewEmailService(repo EmailRepository) *EmailService {
	return &EmailService{repo: repo}
}

func (s *EmailService) GetEmailsByReceiver(ctx context.Context, email string) ([]models.Email, error) {
	emails, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]models.Email, 0)

	for _, e := range emails {
		for _, to := range e.To {
			if to == email {
				result = append(result, e)
				break
			}
		}
	}

	return result, nil
}
