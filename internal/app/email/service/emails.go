package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidReceiver  = errors.New("invalid receiver email")
	ErrNoValidReceivers = errors.New("no valid receivers found")
)

type SendEmailInput struct {
	UserId    int64
	Header    string
	Body      string
	Receivers string
}

type EmailWithReceivers struct {
	Email     *models.Email
	Receivers []int64
}

type Repository interface {
	GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error
	GetUsersByEmails(ctx context.Context, emails []string) (map[string]int64, error)
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

func (s *Service) SendEmail(ctx context.Context, input SendEmailInput) (*models.Email, error) {
	receiverEmails := parseReceivers(input.Receivers)
	if len(receiverEmails) == 0 {
		return nil, ErrInvalidReceiver
	}

	receiverIDs, err := s.repo.GetUsersByEmails(ctx, receiverEmails)
	if err != nil {
		err = mapRepositoryError(err)
		return nil, fmt.Errorf("failed to get receiver IDs: %w", err)
	}

	if len(receiverIDs) == 0 {
		return nil, ErrNoValidReceivers
	}

	email := models.Email{
		SenderID: input.UserId,
		Header:   input.Header,
		Body:     input.Body,
	}

	emailID, err := s.repo.SaveEmail(ctx, email)
	if err != nil {
		err = mapRepositoryError(err)
		return nil, fmt.Errorf("failed to save email: %w", err)
	}
	email.ID = emailID

	var receiverIDsSlice []int64
	for _, id := range receiverIDs {
		receiverIDsSlice = append(receiverIDsSlice, id)
	}

	err = s.repo.AddEmailReceivers(ctx, emailID, receiverIDsSlice)
	if err != nil {
		err = mapRepositoryError(err)
		return nil, fmt.Errorf("failed to add email receivers: %w", err)
	}

	return &email, nil

}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return ErrUserNotFound
	default:
		return err
	}
}

func parseReceivers(receivers string) []string {
	if receivers == "" {
		return nil
	}

	// Разделяем по запятой
	parts := strings.Split(receivers, ",")
	emails := make([]string, 0, len(parts))

	for _, part := range parts {
		email := strings.TrimSpace(part)
		if email != "" {
			emails = append(emails, email)
		}
	}

	return emails
}
