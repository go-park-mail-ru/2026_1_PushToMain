//go:generate mockgen -destination=../../../../mocks/app/email/mock_email_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service Repository

package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	userService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
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
	AddEmailUserWithTx(ctx context.Context, tx *sql.Tx, emailID int64, userID int64, isSender bool) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error)

	GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error)
	MarkEmailAsRead(ctx context.Context, emailID, userID int64) error
	MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error
	GetEmailsCount(ctx context.Context, userID int64) (int, error)
	GetSenderEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error)

	DeleteEmailForReceiver(ctx context.Context, emailID, userID int64) error
	DeleteEmailForSender(ctx context.Context, emailID, userID int64) error

	CheckEmailAccess(ctx context.Context, emailID, userID int64) error
}

type Service struct {
	repo        Repository
	userService *userService.Service
}

func New(repo Repository, userService *userService.Service) *Service {
	return &Service{repo: repo, userService: userService}
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
	ID            int64
	SenderID      int64
	SenderEmail   string
	SenderName    string
	SenderSurname string
	ReceiverList  []string
	Header        string
	Body          string
	CreatedAt     time.Time
	IsRead        bool
}

func (s *Service) GetEmailsByReceiver(ctx context.Context, input GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetEmailsByReceiver(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	total, err := s.repo.GetEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	unreadCount, err := s.repo.GetUnreadEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	resultEmails := make([]EmailResult, len(emails))
	for i, email := range emails {
		user, err := s.userService.GetMe(ctx, email.SenderID)
		if err != nil {
			return nil, userService.MapRepositoryError(err)
		}
		resultEmails[i] = EmailResult{
			ID:            email.ID,
			SenderEmail:   user.Email,
			SenderName:    user.Name,
			SenderSurname: user.Surname,
			ReceiverList:  email.ReceiversEmails,
			Header:        email.Header,
			Body:          email.Body,
			CreatedAt:     email.CreatedAt,
			IsRead:        email.IsRead,
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
		return nil, MapRepositoryError(err)
	}

	total, err := s.repo.GetSenderEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
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
	receiverIDs, err := s.ResolveReceivers(ctx, input.Receivers)
	if err != nil {
		return nil, MapRepositoryError(err)
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
		return nil, MapRepositoryError(err)
	}
	email.ID = emailID
	for _, receiverID := range receiverIDs {
		err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, receiverID, false)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
	}
	err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, email.SenderID, true)
	if err != nil {
		return nil, MapRepositoryError(err)
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
		return MapRepositoryError(err)
	}

	email := models.Email{
		SenderID: input.UserID,
		Header:   forwardEmail.Header,
		Body:     forwardEmail.Body,
	}

	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return MapRepositoryError(err)
	}
	email.ID = emailID

	receiverIDs, err := s.ResolveReceivers(ctx, input.Receivers)
	if err != nil {
		return MapRepositoryError(err)
	}
	for _, receiverID := range receiverIDs {
		err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, receiverID, false)
		if err != nil {
			return MapRepositoryError(err)
		}
	}
	err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, email.SenderID, true)
	if err != nil {
		return MapRepositoryError(err)
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
	SenderEmail     string
	SenderName      string
	SenderSurname   string
	Header          string
	Body            string
	CreatedAt       time.Time
	SenderImagePath string
	ReceiverList    []string
}

func (s *Service) GetEmailByID(ctx context.Context, input GetEmailInput) (*GetEmailResult, error) {

	err := s.repo.CheckEmailAccess(ctx, input.EmailID, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	email, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	user, err := s.userService.GetMe(ctx, email.SenderID)
	if err != nil {
		return nil, userService.MapRepositoryError(err)
	}
	return &GetEmailResult{
		ID:              email.ID,
		SenderID:        email.SenderID,
		SenderEmail:     user.Email,
		SenderName:      user.Name,
		SenderSurname:   user.Surname,
		Header:          email.Header,
		Body:            email.Body,
		CreatedAt:       email.CreatedAt,
		SenderImagePath: email.SenderImagePath,
		ReceiverList:    email.ReceiversEmails,
	}, nil
}

type DeleteEmailInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) DeleteEmailForReceiver(ctx context.Context, input DeleteEmailInput) error {
	exists, err := s.repo.CheckUserEmailExists(ctx, input.EmailID, input.UserID)
	if err != nil {
		return MapRepositoryError(err)
	}

	if !exists {
		return ErrEmailNotFound
	}

	err = s.repo.DeleteEmailForReceiver(ctx, input.EmailID, input.UserID)
	if err != nil {
		return MapRepositoryError(err)
	}

	return nil
}

func (s *Service) DeleteEmailForSender(ctx context.Context, input DeleteEmailInput) error {
	err := s.repo.DeleteEmailForSender(ctx, input.EmailID, input.UserID)
	if err != nil {
		return MapRepositoryError(err)
	}

	return nil
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID []int64
}

func (s *Service) MarkEmailAsRead(ctx context.Context, input MarkAsReadInput) error {
	for _, emailID := range input.EmailID {
		if err := s.repo.MarkEmailAsRead(ctx, emailID, input.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}

	return nil
}

func (s *Service) MarkEmailAsUnRead(ctx context.Context, input MarkAsReadInput) error {
	for _, emailID := range input.EmailID {
		if err := s.repo.MarkEmailAsUnRead(ctx, emailID, input.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}

	return nil
}

func (s *Service) ResolveReceivers(ctx context.Context, receiverEmails []string) ([]int64, error) {
	if len(receiverEmails) == 0 {
		return nil, ErrNoValidReceivers
	}

	users, err := s.repo.GetUsersByEmails(ctx, receiverEmails)
	if err != nil {
		return nil, MapRepositoryError(err)
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

func MapRepositoryError(err error) error {
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
