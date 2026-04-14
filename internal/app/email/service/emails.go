package service

import (
	"context"
	"database/sql"
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
	ErrTransaction      = errors.New("transaction fail")
	ErrConflict         = errors.New("conflict")
	ErrBadRequest       = errors.New("bad request")
)

type Repository interface {
	SaveEmailWithTx(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error)
	AddEmailReceiversWithTx(ctx context.Context, tx *sql.Tx, emailID int64, receiverIDs []int64) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error)

	GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	AddEmailReceivers(ctx context.Context, emailID int64, receiverIDs []int64) error
	GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error)
	MarkEmailAsRead(ctx context.Context, emailID, userID int64) error
	MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error
	GetEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUserEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error)

	DeleteEmailForReceiver(ctx context.Context, emailID, userID int64) error
	DeleteEmailForSender(ctx context.Context, emailID, userID int64) error

	CheckEmailAccess(ctx context.Context, emailID, userID int64) error
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
	Emails      []EmailResult
	Limit       int
	Offset      int
	Total       int
	UnreadCount int
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

	total, err := s.repo.GetEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	unreadCount, err := s.repo.GetUnreadEmailsCount(ctx, input.UserID)
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
		Emails:      resultEmails,
		Limit:       input.Limit,
		Offset:      input.Offset,
		Total:       total,
		UnreadCount: unreadCount,
	}, nil
}

type GetMyEmailsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetMyEmailsResult struct {
	Emails []MyEmailResult
	Limit  int
	Offset int
	Total  int
}

type MyEmailResult struct {
	ID              int64
	SenderID        int64
	Header          string
	Body            string
	CreatedAt       time.Time
	IsRead          bool
	ReceiversEmails []string
}

func (s *Service) GetEmailsBySender(ctx context.Context, input GetMyEmailsInput) (*GetMyEmailsResult, error) {
	emails, err := s.repo.GetEmailsBySender(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	total, err := s.repo.GetUserEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	resultEmails := make([]MyEmailResult, len(emails))
	for i, email := range emails {
		resultEmails[i] = MyEmailResult{
			ID:              email.ID,
			SenderID:        email.SenderID,
			Header:          email.Header,
			Body:            email.Body,
			CreatedAt:       email.CreatedAt,
			IsRead:          email.IsRead,
			ReceiversEmails: email.ReceiversEmails,
		}
	}

	return &GetMyEmailsResult{
		Emails: resultEmails,
		Limit:  input.Limit,
		Offset: input.Offset,
		Total:  total,
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

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, ErrTransaction
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = ErrTransaction
			}
		}
	}()

	email := models.Email{
		SenderID: input.UserId,
		Header:   input.Header,
		Body:     input.Body,
	}

	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	email.ID = emailID

	err = s.repo.AddEmailReceiversWithTx(ctx, tx, emailID, receiverIDs)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
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

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return ErrTransaction
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = ErrTransaction
			}
		}
	}()

	getEmail := GetEmailInput{
		UserID:  input.UserID,
		EmailID: input.EmailID,
	}

	forwardEmail, err := s.GetEmailByID(ctx, getEmail)
	if err != nil {
		return mapRepositoryError(err)
	}

	email := models.Email{
		SenderID: input.UserID,
		Header:   forwardEmail.Header,
		Body:     forwardEmail.Body,
	}

	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return mapRepositoryError(err)
	}
	email.ID = emailID

	receiverIDs, err := s.resolveReceivers(ctx, input.Receivers)
	if err != nil {
		return mapRepositoryError(err)
	}

	err = s.repo.AddEmailReceiversWithTx(ctx, tx, emailID, receiverIDs)
	if err != nil {
		return mapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return ErrTransaction
	}

	return nil

}

type GetEmailInput struct {
	UserID  int64
	EmailID int64
}

type GetEmailResult struct {
	ID              int64
	SenderID        int64
	Header          string
	Body            string
	CreatedAt       time.Time
	SenderImagePath string
}

func (s *Service) GetEmailByID(ctx context.Context, input GetEmailInput) (*GetEmailResult, error) {

	err := s.repo.CheckEmailAccess(ctx, input.EmailID, input.UserID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	return &GetEmailResult{
		ID:              email.ID,
		SenderID:        email.SenderID,
		Header:          email.Header,
		Body:            email.Body,
		CreatedAt:       email.CreatedAt,
		SenderImagePath: email.SenderImagePath,
	}, nil
}

type DeleteEmailInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) DeleteEmailForReceiver(ctx context.Context, input DeleteEmailInput) error {
	exists, err := s.repo.CheckUserEmailExists(ctx, input.EmailID, input.UserID)
	if err != nil {
		return mapRepositoryError(err)
	}

	if !exists {
		return ErrEmailNotFound
	}

	err = s.repo.DeleteEmailForReceiver(ctx, input.EmailID, input.UserID)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *Service) DeleteEmailForSender(ctx context.Context, input DeleteEmailInput) error {
	err := s.repo.DeleteEmailForSender(ctx, input.EmailID, input.UserID)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) MarkEmailAsRead(ctx context.Context, input MarkAsReadInput) error {
	err := s.repo.MarkEmailAsRead(ctx, input.EmailID, input.UserID)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *Service) MarkEmailAsUnRead(ctx context.Context, input MarkAsReadInput) error {
	err := s.repo.MarkEmailAsUnRead(ctx, input.EmailID, input.UserID)
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
	case errors.Is(err, repository.ErrDuplicate):
		return ErrConflict
	case errors.Is(err, repository.ErrForeignKey):
		return ErrBadRequest
	case errors.Is(err, repository.ErrUserNotFound):
		return ErrUserNotFound
	case errors.Is(err, repository.ErrReceiverAdd):
		return ErrNoValidReceivers
	case errors.Is(err, repository.ErrMailNotFound):
		return ErrEmailNotFound
	case errors.Is(err, repository.ErrAccessDenied):
		return ErrAccessDenied
	default:
		return err
	}
}
