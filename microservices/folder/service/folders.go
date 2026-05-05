package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/repository"
	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"
)

var (
	ErrFolderAlreadyExists = errors.New("folder with this name already exists")
	ErrMaxFoldersReached   = errors.New("maximum number of folders reached (max 10 custom folders)")
	ErrFolderNotFound      = errors.New("folder not found")
	ErrAccessDenied        = errors.New("access error")
	ErrEmptyEmailsList     = errors.New("no emails to add error")
)

const MaxCustomFolders = 10

type Repository interface {
	CreateFolder(ctx context.Context, folder models.Folder) (int64, error)
	GetFolderByName(ctx context.Context, userID int64, name string) (*models.Folder, error)
	CountUserFolders(ctx context.Context, userID int64) (int, error)
	GetFolderByID(ctx context.Context, folderID int64) (*models.Folder, error)
	UpdateFolderName(ctx context.Context, folderID int64, newName string) error
	CountEmailsInFolder(ctx context.Context, folderID int64) (int, error)
	AddEmailToFolder(ctx context.Context, folderID, emailID int64) error
	DeleteEmailFromFolder(ctx context.Context, folderID, emailID int64) error
	GetFolderEmailIDs(ctx context.Context, folderID int64, limit, offset int) ([]int64, error)
	DeleteFolder(ctx context.Context, folderID, userID int64) error
}

type Service struct {
	repo        Repository
	emailClient EmailClient
}
type EmailClient interface {
	GetEmailByID(
		ctx context.Context,
		emailID,
		userID int64,
	) (*emailpb.Email, error)

	CheckEmailAccess(
		ctx context.Context,
		emailID,
		userID int64,
	) (bool, error)
	GetEmailsByIDs(ctx context.Context, emailIDs []int64, userID int64) (*emailpb.GetEmailsByIdsResponse, error)
}

func New(
	repo Repository,
	emailClient EmailClient,
) *Service {

	return &Service{
		repo:        repo,
		emailClient: emailClient,
	}
}

type CreateNewFolderInput struct {
	UserId     int64
	FolderName string
}

type CreateNewFolderResult struct {
	ID int64
}

func (s *Service) CreateNewFolder(ctx context.Context, input CreateNewFolderInput) (*CreateNewFolderResult, error) {
	count, err := s.repo.CountUserFolders(ctx, input.UserId)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	if count >= MaxCustomFolders {
		return nil, ErrMaxFoldersReached
	}

	folder := models.Folder{
		UserID: input.UserId,
		Name:   input.FolderName,
	}

	folderID, err := s.repo.CreateFolder(ctx, folder)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	return &CreateNewFolderResult{
		ID: folderID,
	}, nil
}

type ChangeFolderNameInput struct {
	UserID     int64
	FolderID   int64
	FolderName string
}

func (s *Service) ChangeFolderName(ctx context.Context, input ChangeFolderNameInput) error {
	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return MapRepositoryError(err)
	}

	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	err = s.repo.UpdateFolderName(ctx, input.FolderID, input.FolderName)
	if err != nil {
		return MapRepositoryError(err)
	}

	return nil
}

type GetEmailsFromFolderInput struct {
	UserID   int64
	FolderID int64
	Limit    int
	Offset   int
}

type GetEmailsFromFolderResult struct {
	Emails      []EmailFromFolderResult
	Limit       int
	Offset      int
	Total       int
	UnreadCount int
}

type EmailFromFolderResult struct {
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

func (s *Service) GetEmailsFromFolder(ctx context.Context, input GetEmailsFromFolderInput) (*GetEmailsFromFolderResult, error) {

	folder, err := s.repo.GetFolderByID(
		ctx,
		input.FolderID,
	)

	if err != nil {
		return nil, MapRepositoryError(err)
	}

	if folder.UserID != input.UserID {
		return nil, ErrAccessDenied
	}

	emailIDs, err := s.repo.GetFolderEmailIDs(
		ctx,
		input.FolderID,
		input.Limit,
		input.Offset,
	)

	if err != nil {
		return nil, MapRepositoryError(err)
	}

	emailResp, err := s.emailClient.GetEmailsByIDs(
		ctx,
		emailIDs,
		input.UserID,
	)

	if err != nil {
		return nil, err
	}

	total, err := s.repo.CountEmailsInFolder(
		ctx,
		input.FolderID,
	)

	if err != nil {
		return nil, MapRepositoryError(err)
	}

	resultEmails := make(
		[]EmailFromFolderResult,
		len(emailResp.Emails),
	)

	for i, email := range emailResp.Emails {

		resultEmails[i] = EmailFromFolderResult{
			ID:            email.Id,
			SenderEmail:   email.SenderEmail,
			SenderName:    email.SenderName,
			SenderSurname: email.SenderSurname,
			ReceiverList:  email.ReceiverList,
			Header:        email.Header,
			Body:          email.Body,
			CreatedAt:     email.CreatedAt.AsTime(),
			IsRead:        email.IsRead,
		}
	}

	return &GetEmailsFromFolderResult{
		Emails:      resultEmails,
		Limit:       input.Limit,
		Offset:      input.Offset,
		Total:       total,
		UnreadCount: int(emailResp.UnreadCount),
	}, nil
}

type AddEmailsInFolderInput struct {
	UserID   int64
	FolderID int64
	EmailsID []int64
}

func (s *Service) AddEmailsInFolder(ctx context.Context, input AddEmailsInFolderInput) error {

	if len(input.EmailsID) == 0 {
		return ErrEmptyEmailsList
	}

	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return MapRepositoryError(err)
	}
	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	for _, emailID := range input.EmailsID {
		hasAccess, err := s.emailClient.CheckEmailAccess(ctx, emailID, input.UserID)
		if err != nil {
			return MapRepositoryError(err)
		}
		if !hasAccess {
			return MapRepositoryError(err)
		}

		if err := s.repo.AddEmailToFolder(ctx, input.FolderID, emailID); err != nil {
			return MapRepositoryError(err)
		}
	}

	return nil
}

type DeleteEmailsFromFolderInput struct {
	UserID   int64
	FolderID int64
	EmailsID []int64
}

func (s *Service) DeleteEmailsFromFolder(ctx context.Context, input DeleteEmailsFromFolderInput) error {
	if len(input.EmailsID) == 0 {
		return ErrEmptyEmailsList
	}

	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return MapRepositoryError(err)
	}
	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	// Удаляем каждое письмо из папки
	for _, emailID := range input.EmailsID {
		if err := s.repo.DeleteEmailFromFolder(ctx, input.FolderID, emailID); err != nil {
			return MapRepositoryError(err)
		}
	}

	return nil
}

type DeleteFolderInput struct {
	UserID   int64
	FolderID int64
}

func (s *Service) DeleteFolder(ctx context.Context, input DeleteFolderInput) error {
	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return MapRepositoryError(err)
	}
	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	err = s.repo.DeleteFolder(ctx, input.FolderID, input.UserID)
	if err != nil {
		return MapRepositoryError(err)
	}

	return nil
}

func MapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrDuplicate):
		return ErrFolderAlreadyExists
	case errors.Is(err, repository.ErrFolderNotFound):
		return ErrFolderNotFound
	default:
		return err
	}
}
