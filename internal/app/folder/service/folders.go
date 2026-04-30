package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/repository"
)

var (
	ErrFolderNameEmpty     = errors.New("folder name cannot be empty")
	ErrFolderNameTooLong   = errors.New("folder name too long (max 255 characters)")
	ErrFolderNameInvalid   = errors.New("folder name contains invalid characters")
	ErrFolderAlreadyExists = errors.New("folder with this name already exists")
	ErrMaxFoldersReached   = errors.New("maximum number of folders reached (max 10 custom folders)")
	ErrFolderNotFound      = errors.New("folder not found")
	ErrAccessDenied        = errors.New("access error")
	ErrEmptyEmailsList     = errors.New("no emails to add error")
)

const (
	MaxFolderNameLength = 255
	MaxCustomFolders    = 10
)

type Repository interface {
	CreateFolder(ctx context.Context, folder models.Folder) (int64, error)
	GetFolderByName(ctx context.Context, userID int64, name string) (*models.Folder, error)
	CountUserFolders(ctx context.Context, userID int64) (int, error)
	GetFolderByID(ctx context.Context, folderID int64) (*models.Folder, error)
	UpdateFolderName(ctx context.Context, folderID int64, newName string) error
	GetEmailsFromFolder(ctx context.Context, folderID int64, limit, offset int) ([]models.EmailFromFolder, error)
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
	if input.FolderName == "" {
		return nil, ErrFolderNameEmpty
	}
	if len(input.FolderName) > MaxFolderNameLength {
		return nil, ErrFolderNameTooLong
	}
	validName := regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9\s\-_]+$`)
	if !validName.MatchString(input.FolderName) {
		return nil, ErrFolderNameInvalid
	}

	count, err := s.repo.CountUserFolders(ctx, input.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to count user folders: %w", err)
	}

	if count >= MaxCustomFolders {
		return nil, ErrMaxFoldersReached
	}

	existing, err := s.repo.GetFolderByName(ctx, input.UserId, input.FolderName)
	err = MapRepositoryError(err)
	if err != nil && !errors.Is(err, ErrFolderNotFound) {
		return nil, fmt.Errorf("failed to check existing folder: %w", err)
	}
	if existing != nil {
		return nil, ErrFolderAlreadyExists
	}

	folder := models.Folder{
		UserID: input.UserId,
		Name:   input.FolderName,
	}

	folderID, err := s.repo.CreateFolder(ctx, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
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
	if input.FolderName == "" {
		return ErrFolderNameEmpty
	}

	if len(input.FolderName) > MaxFolderNameLength {
		return ErrFolderNameTooLong
	}

	validName := regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9\s\-_]+$`)
	if !validName.MatchString(input.FolderName) {
		return ErrFolderNameInvalid
	}

	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return fmt.Errorf("failed to get folder: %w", err)
	}

	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	existingFolder, err := s.repo.GetFolderByName(ctx, input.UserID, input.FolderName)
	err = MapRepositoryError(err)
	if err != nil && !errors.Is(err, ErrFolderNotFound) {
		return fmt.Errorf("failed to check existing folder: %w", err)
	}
	if existingFolder != nil && existingFolder.ID != input.FolderID {
		return ErrFolderAlreadyExists
	}

	err = s.repo.UpdateFolderName(ctx, input.FolderID, input.FolderName)
	if err != nil {
		return fmt.Errorf("failed to update folder name: %w", err)
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
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 20
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	folder, err := s.repo.GetFolderByID(ctx, input.FolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	if folder.UserID != input.UserID {
		return nil, ErrAccessDenied
	}

	emails, err := s.repo.GetEmailsFromFolder(ctx, input.FolderID, input.Limit, input.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails from folder: %w", err)
	}

	total, err := s.repo.CountEmailsInFolder(ctx, input.FolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to count emails in folder: %w", err)
	}

	unreadCount, err := s.repo.CountUnreadEmailsInFolder(ctx, input.FolderID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count unread emails: %w", err)
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
		return fmt.Errorf("failed to get folder: %w", err)
	}
	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	for _, emailID := range input.EmailsID {
		hasAccess, err := s.repo.CheckEmailAccess(ctx, emailID, input.UserID)
		if err != nil {
			return fmt.Errorf("failed to check email access for email %d: %w", emailID, err)
		}
		if !hasAccess {
			return fmt.Errorf("email %d not found or access denied", emailID)
		}

		if err := s.repo.AddEmailToFolder(ctx, input.FolderID, emailID); err != nil {
			return fmt.Errorf("failed to add email %d to folder: %w", emailID, err)
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
		return fmt.Errorf("failed to get folder: %w", err)
	}
	if folder.UserID != input.UserID {
		return ErrAccessDenied
	}

	// Удаляем каждое письмо из папки
	for _, emailID := range input.EmailsID {
		if err := s.repo.DeleteEmailFromFolder(ctx, input.FolderID, emailID); err != nil {
			return fmt.Errorf("failed to delete email %d from folder: %w", emailID, err)
		}
	}

	return nil
}

func MapRepositoryError(err error) error {
	switch {
	case errors.Is(err, repository.ErrFolderNotFound):
		return ErrFolderNotFound
	default:
		return err
	}
}
