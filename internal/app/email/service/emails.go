package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailNotFound    = errors.New("email not found")
	ErrNoValidReceivers = errors.New("no valid receivers found")
	ErrAccessDenied     = errors.New("don't have access to this email")
)

type Repository interface {
	GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error
	GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.Email, error)
	MarkEmailAsRead(ctx context.Context, emailID, userID int64) error
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

type GetEmailsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetEmailsResult struct {
	Emails []EmailResult
	Limit  int
	Offset int
}

type EmailResult struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	CreatedAt time.Time
	IsRead    bool
}

func (s *Service) GetEmailsByReceiver(ctx context.Context, input GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetEmailsByReceiver(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	resultEmails := make([]EmailResult, len(emails))
	for i, email := range emails {
		resultEmails[i] = EmailResult{
			ID:        email.ID,
			SenderID:  email.SenderID,
			Header:    email.Header,
			Body:      email.Body,
			CreatedAt: email.CreatedAt,
			IsRead:    email.IsRead,
		}
	}

	return &GetEmailsResult{
		Emails: resultEmails,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
}

type SendEmailInput struct {
	UserId    int64
	Header    string
	Body      string
	Receivers []string
}

type SendEmailResult struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	CreatedAt time.Time
}

func (s *Service) SendEmail(ctx context.Context, input SendEmailInput) (*SendEmailResult, error) {
	receiverIDs, err := s.resolveReceivers(ctx, input.Receivers)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	email := models.Email{
		SenderID: input.UserId,
		Header:   input.Header,
		Body:     input.Body,
	}

	emailID, err := s.repo.SaveEmail(ctx, email)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	email.ID = emailID

	err = s.repo.AddEmailReceivers(ctx, emailID, receiverIDs)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	emailResult := SendEmailResult{
		ID:        email.ID,
		SenderID:  email.SenderID,
		Header:    email.Header,
		Body:      email.Body,
		CreatedAt: email.CreatedAt,
	}

	return &emailResult, nil
}

type ForwardEmailInput struct {
	UserID    int64
	EmailID   int64
	Receivers []string
}

func (s *Service) ForwardEmail(ctx context.Context, input ForwardEmailInput) error {
	receiverIDs, err := s.resolveReceivers(ctx, input.Receivers)
	if err != nil {
		return mapRepositoryError(err)
	}
	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return mapRepositoryError(err)
	}

	if email.SenderID != input.UserID {
		return ErrAccessDenied
	}

	err = s.repo.AddEmailReceivers(ctx, input.EmailID, receiverIDs)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil

}

type GetEmailInput struct {
	UserID  int64
	EmailID int64
}

type GetEmailResult struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	CreatedAt time.Time
}

func (s *Service) GetEmailByID(ctx context.Context, input GetEmailInput) (*GetEmailResult, error) {
	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if email.SenderID != input.UserID {
		return nil, ErrAccessDenied
	}

	return &GetEmailResult{
		ID:        email.ID,
		SenderID:  email.SenderID,
		Header:    email.Header,
		Body:      email.Body,
		CreatedAt: email.CreatedAt,
	}, nil
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) MarkEmailAsRead(ctx context.Context, input MarkAsReadInput) error {
	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return mapRepositoryError(err)
	}

	if email.SenderID != input.UserID {
		return ErrAccessDenied
	}

	err = s.repo.MarkEmailAsRead(ctx, input.EmailID, input.UserID)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *Service) resolveReceivers(ctx context.Context, receiverEmails []string) ([]int64, error) {
	if len(receiverEmails) == 0 {
		return nil, ErrNoValidReceivers
	}

	users, err := s.repo.GetUsersByEmails(ctx, receiverEmails)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if len(users) == 0 {
		return nil, ErrNoValidReceivers
	}

	receiverIDs := make([]int64, len(users))
	for i, user := range users {
		receiverIDs[i] = user.ID
	}

	return receiverIDs, nil
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return ErrUserNotFound
	case errors.Is(err, repository.ErrReceiverAdd):
		return ErrNoValidReceivers
	case errors.Is(err, repository.ErrMailNotFound):
		return ErrEmailNotFound
	default:
		return err
	}
}
