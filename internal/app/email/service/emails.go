package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrNoValidReceivers = errors.New("no valid receivers found")
	ErrAccessDenied     = errors.New("don't have access to this email")
)

type EmailWithReceivers struct {
	Email     *models.Email
	Receivers []int64
}

type Repository interface {
	GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error
	GetUsersByEmails(ctx context.Context, emails []string) (map[string]int64, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.Email, error)
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

type SendEmailInput struct {
	UserId    int64
	Header    string
	Body      string
	Receivers []string
}

func (s *Service) SendEmail(ctx context.Context, input SendEmailInput) (*models.Email, error) {
	receiverIDs, err := s.repo.GetUsersByEmails(ctx, input.Receivers)
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

type ForwardEmailInput struct {
	UserID    int64
	EmailID   int64
	Receivers []string
}

func (s *Service) ForwardEmail(ctx context.Context, input ForwardEmailInput) error {
	receiverIDs, err := s.repo.GetUsersByEmails(ctx, input.Receivers)
	if err != nil {
		err = mapRepositoryError(err)
		return fmt.Errorf("failed to get receiver IDs: %w", err)
	}

	if len(receiverIDs) == 0 {
		return ErrNoValidReceivers
	}

	var receiverIDsSlice []int64
	for _, id := range receiverIDs {
		receiverIDsSlice = append(receiverIDsSlice, id)
	}

	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		err = mapRepositoryError(err)
		return fmt.Errorf("failed to find email: %w", err)
	}

	if email.SenderID != input.UserID {
		return ErrAccessDenied
	}

	err = s.repo.AddEmailReceivers(ctx, input.EmailID, receiverIDsSlice)
	if err != nil {
		err = mapRepositoryError(err)
		return fmt.Errorf("failed to add email receivers: %w", err)
	}

	return nil

}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return ErrUserNotFound
	default:
		return err
	}
}
