package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/repository"
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
	GetEmailsFromFolder(ctx context.Context, folderID, userID int64, limit, offset int) ([]models.EmailFromFolder, error)
	CountEmailsInFolder(ctx context.Context, folderID int64) (int, error)
	CountUnreadEmailsInFolder(ctx context.Context, folderID, userID int64) (int, error)
	CheckEmailAccess(ctx context.Context, emailID, userID int64) (bool, error)
	AddEmailToFolder(ctx context.Context, folderID, emailID int64) error
	DeleteEmailFromFolder(ctx context.Context, folderID, emailID int64) error
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
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
	IsFavorite    bool
}

func (s *Service) GetEmailsFromFolder(ctx context.Context, input GetEmailsFromFolderInput) (*GetEmailsFromFolderResult, error) {
	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if folder.UserID != input.UserID {
		return nil, ErrAccessDenied
	}

	emails, err := s.repo.GetEmailsFromFolder(ctx, input.FolderID, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	total, err := s.repo.CountEmailsInFolder(ctx, input.FolderID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	unreadCount, err := s.repo.CountUnreadEmailsInFolder(ctx, input.FolderID, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	resultEmails := make([]EmailFromFolderResult, len(emails))
	for i, email := range emails {
		resultEmails[i] = EmailFromFolderResult{
			ID:            email.ID,
			SenderEmail:   email.SenderEmail,
			SenderName:    email.SenderName,
			SenderSurname: email.SenderSurname,
			ReceiverList:  email.ReceiverList,
			Header:        email.Header,
			Body:          email.Body,
			CreatedAt:     email.CreatedAt,
			IsRead:        email.IsRead,
			IsFavorite:    email.IsFavorite,
		}
	}

	return &GetEmailsFromFolderResult{
		Emails:      resultEmails,
		Limit:       input.Limit,
		Offset:      input.Offset,
		Total:       total,
		UnreadCount: unreadCount,
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
		hasAccess, err := s.repo.CheckEmailAccess(ctx, emailID, input.UserID)
		if err != nil {
			return MapRepositoryError(err)
		}
		if !hasAccess {
			return ErrAccessDenied
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

	for _, emailID := range input.EmailsID {
		if err := s.repo.DeleteEmailFromFolder(ctx, input.FolderID, emailID); err != nil {
			return MapRepositoryError(err)
		}
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
