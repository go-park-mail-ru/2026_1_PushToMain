//go:generate mockgen -destination=../../../../mocks/app/email/mock_email_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service Repository

package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	folderModels "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/models"
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
	ErrInvalidFolder    = errors.New("invalid folder tag")
	ErrFolderNotFound   = errors.New("folder not found")
	ErrDraftNotReady    = errors.New("draft is not ready to be sent")
	ErrDraftValidation  = errors.New("draft must contain at least one of: header, body, receivers")
	ErrDraftsLimit      = errors.New("drafts limit reached")
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
	CheckEmailAccess(ctx context.Context, emailID, userID int64) error

	GetUserEmailFlags(ctx context.Context, emailID, userID int64, isSender bool) (*models.UserEmail, error)
	SoftDeleteUserEmail(ctx context.Context, emailID, userID int64, isSender bool) error
	HardDeleteUserEmail(ctx context.Context, emailID, userID int64, isSender bool) error
	RestoreFromTrash(ctx context.Context, emailID, userID int64) error

	SetStarred(ctx context.Context, emailID, userID int64, starred bool) error
	MarkSenderAsSpam(ctx context.Context, emailID, userID int64) (int64, error)
	MoveToTrash(ctx context.Context, emailID, userID int64) error

	GetSpamEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetTrashEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetSpamEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadSpamCount(ctx context.Context, userID int64) (int, error)
	GetTrashEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadTrashCount(ctx context.Context, userID int64) (int, error)

	CountDraftsByUser(ctx context.Context, userID int64) (int, error)
	CreateDraft(ctx context.Context, draft models.Draft) (int64, error)
	UpdateDraft(ctx context.Context, draft models.Draft) error
	GetDraftByID(ctx context.Context, draftID, userID int64) (*models.Draft, error)
	GetDrafts(ctx context.Context, userID int64, limit, offset int) ([]models.Draft, error)
	DeleteDraft(ctx context.Context, draftID, userID int64) error
	MarkDraftAsSentTx(ctx context.Context, tx *sql.Tx, draftID, userID int64) error
}

type FolderRepository interface {
	GetFolderByName(ctx context.Context, userID int64, name string) (*folderModels.Folder, error)
	AddEmailToFolder(ctx context.Context, folderID, emailID int64) error
}

type DraftsConfig struct {
	MaxPerUser int
}

type Service struct {
	repo        Repository
	folderRepo  FolderRepository
	userService *userService.Service
	drafts      DraftsConfig
}

func New(
	repo Repository,
	folderRepo FolderRepository,
	userService *userService.Service,
	drafts DraftsConfig,
) *Service {
	return &Service{
		repo:        repo,
		folderRepo:  folderRepo,
		userService: userService,
		drafts:      drafts,
	}
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
	IsStarred     bool
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
	unread, err := s.repo.GetUnreadEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, input.Limit, input.Offset, total, unread)
}

func (s *Service) GetSpamEmails(ctx context.Context, input GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetSpamEmails(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetSpamEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	unread, err := s.repo.GetUnreadSpamCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, input.Limit, input.Offset, total, unread)
}

func (s *Service) GetTrashEmails(ctx context.Context, input GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetTrashEmails(ctx, input.UserID, input.Limit, input.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetTrashEmailsCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	unread, err := s.repo.GetUnreadTrashCount(ctx, input.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, input.Limit, input.Offset, total, unread)
}

func (s *Service) buildEmailsResult(
	ctx context.Context, emails []models.EmailWithMetadata,
	limit, offset, total, unread int,
) (*GetEmailsResult, error) {
	out := make([]EmailResult, len(emails))
	for i, em := range emails {
		user, err := s.userService.GetMe(ctx, em.SenderID)
		if err != nil {
			return nil, userService.MapRepositoryError(err)
		}
		out[i] = EmailResult{
			ID:            em.ID,
			SenderID:      em.SenderID,
			SenderEmail:   user.Email,
			SenderName:    user.Name,
			SenderSurname: user.Surname,
			ReceiverList:  em.ReceiversEmails,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsRead:        em.IsRead,
			IsStarred:     em.IsStarred,
		}
	}
	return &GetEmailsResult{
		Emails:      out,
		Limit:       limit,
		Offset:      offset,
		Total:       total,
		UnreadCount: unread,
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
	out := make([]MyEmailResult, len(emails))
	for i, em := range emails {
		out[i] = MyEmailResult{
			ID:              em.ID,
			SenderID:        em.SenderID,
			Header:          em.Header,
			Body:            em.Body,
			CreatedAt:       em.CreatedAt,
			IsRead:          em.IsRead,
			ReceiversEmails: em.ReceiversEmails,
		}
	}
	return &GetMyEmailsResult{
		Emails: out,
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
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	email := models.Email{SenderID: input.UserId, Header: input.Header, Body: input.Body}
	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	email.ID = emailID

	for _, rid := range receiverIDs {
		if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, rid, false); err != nil {
			return nil, MapRepositoryError(err)
		}
	}
	if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, email.SenderID, true); err != nil {
		return nil, MapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	return &SendEmailResult{
		ID:        email.ID,
		SenderID:  email.SenderID,
		Header:    email.Header,
		Body:      email.Body,
		CreatedAt: email.CreatedAt,
	}, nil
}

type ForwardEmailInput struct {
	UserID    int64
	EmailID   int64
	Receivers []string
}

func (s *Service) ForwardEmail(ctx context.Context, input ForwardEmailInput) error {
	src, err := s.GetEmailByID(ctx, GetEmailInput{UserID: input.UserID, EmailID: input.EmailID})
	if err != nil {
		return MapRepositoryError(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	email := models.Email{SenderID: input.UserID, Header: src.Header, Body: src.Body}
	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return MapRepositoryError(err)
	}

	receiverIDs, err := s.ResolveReceivers(ctx, input.Receivers)
	if err != nil {
		return MapRepositoryError(err)
	}
	for _, rid := range receiverIDs {
		if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, rid, false); err != nil {
			return MapRepositoryError(err)
		}
	}
	if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, input.UserID, true); err != nil {
		return MapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return ErrTransaction
	}
	committed = true
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
	if err := s.repo.CheckEmailAccess(ctx, input.EmailID, input.UserID); err != nil {
		return nil, MapRepositoryError(err)
	}
	em, err := s.repo.GetEmailByID(ctx, input.EmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	user, err := s.userService.GetMe(ctx, em.SenderID)
	if err != nil {
		return nil, userService.MapRepositoryError(err)
	}
	return &GetEmailResult{
		ID:              em.ID,
		SenderID:        em.SenderID,
		SenderEmail:     user.Email,
		SenderName:      user.Name,
		SenderSurname:   user.Surname,
		Header:          em.Header,
		Body:            em.Body,
		CreatedAt:       em.CreatedAt,
		SenderImagePath: em.SenderImagePath,
		ReceiverList:    em.ReceiversEmails,
	}, nil
}

type DeleteEmailInput struct {
	UserID  int64
	EmailID int64
}

func (s *Service) DeleteEmailForReceiver(ctx context.Context, input DeleteEmailInput) error {
	return s.deleteWithTwoStages(ctx, input.EmailID, input.UserID, false)
}

func (s *Service) DeleteEmailForSender(ctx context.Context, input DeleteEmailInput) error {
	return s.deleteWithTwoStages(ctx, input.EmailID, input.UserID, true)
}

func (s *Service) deleteWithTwoStages(ctx context.Context, emailID, userID int64, isSender bool) error {
	flags, err := s.repo.GetUserEmailFlags(ctx, emailID, userID, isSender)
	if err != nil {
		return MapRepositoryError(err)
	}
	if flags.IsDeleted {
		return MapRepositoryError(s.repo.HardDeleteUserEmail(ctx, emailID, userID, isSender))
	}
	return MapRepositoryError(s.repo.SoftDeleteUserEmail(ctx, emailID, userID, isSender))
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID []int64
}

func (s *Service) MarkEmailAsRead(ctx context.Context, input MarkAsReadInput) error {
	for _, id := range input.EmailID {
		if err := s.repo.MarkEmailAsRead(ctx, id, input.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}

func (s *Service) MarkEmailAsUnRead(ctx context.Context, input MarkAsReadInput) error {
	for _, id := range input.EmailID {
		if err := s.repo.MarkEmailAsUnRead(ctx, id, input.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}

const (
	FolderTagSpam     = "spam"
	FolderTagFavorite = "favorite"
	FolderTagTrash    = "trash"
	customFolderPfx   = "folder-"
)

type ChangeFolderInput struct {
	UserID  int64
	EmailID int64
	Folder  string
}

func (s *Service) ChangeFolder(ctx context.Context, in ChangeFolderInput) error {
	if err := s.repo.CheckEmailAccess(ctx, in.EmailID, in.UserID); err != nil {
		return MapRepositoryError(err)
	}

	switch in.Folder {
	case FolderTagSpam:
		if _, err := s.repo.MarkSenderAsSpam(ctx, in.EmailID, in.UserID); err != nil {
			return MapRepositoryError(err)
		}
		return nil

	case FolderTagFavorite:
		if err := s.repo.SetStarred(ctx, in.EmailID, in.UserID, true); err != nil {
			return MapRepositoryError(err)
		}
		return nil

	case FolderTagTrash:
		if err := s.repo.MoveToTrash(ctx, in.EmailID, in.UserID); err != nil {
			return MapRepositoryError(err)
		}
		return nil

	default:
		if !strings.HasPrefix(in.Folder, customFolderPfx) {
			return ErrInvalidFolder
		}
		folder, err := s.folderRepo.GetFolderByName(ctx, in.UserID, in.Folder)
		if err != nil {
			return ErrFolderNotFound
		}
		if err := s.folderRepo.AddEmailToFolder(ctx, folder.ID, in.EmailID); err != nil {
			return MapRepositoryError(err)
		}
		return nil
	}
}

func (s *Service) RestoreFromTrash(ctx context.Context, in ChangeFolderInput) error {
	if err := s.repo.CheckEmailAccess(ctx, in.EmailID, in.UserID); err != nil {
		return MapRepositoryError(err)
	}
	if err := s.repo.RestoreFromTrash(ctx, in.EmailID, in.UserID); err != nil {
		return MapRepositoryError(err)
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
	out := make([]int64, len(users))
	for i, u := range users {
		out[i] = u.ID
	}
	return out, nil
}

type DraftResult struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	Receivers []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateDraftInput struct {
	UserID    int64
	Header    string
	Body      string
	Receivers []string
}

func validateDraftPayload(header, body string, receivers []string) error {
	if header == "" && body == "" && len(receivers) == 0 {
		return ErrDraftValidation
	}
	return nil
}

func (s *Service) CreateDraft(ctx context.Context, in CreateDraftInput) (*DraftResult, error) {
	if err := validateDraftPayload(in.Header, in.Body, in.Receivers); err != nil {
		return nil, err
	}
	if s.drafts.MaxPerUser > 0 {
		count, err := s.repo.CountDraftsByUser(ctx, in.UserID)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
		if count >= s.drafts.MaxPerUser {
			return nil, ErrDraftsLimit
		}
	}

	id, err := s.repo.CreateDraft(ctx, models.Draft{
		SenderID: in.UserID, Header: in.Header, Body: in.Body, Receivers: in.Receivers,
	})
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	saved, err := s.repo.GetDraftByID(ctx, id, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(saved), nil
}

type UpdateDraftInput struct {
	UserID    int64
	DraftID   int64
	Header    string
	Body      string
	Receivers []string
}

func (s *Service) UpdateDraft(ctx context.Context, in UpdateDraftInput) (*DraftResult, error) {
	if err := validateDraftPayload(in.Header, in.Body, in.Receivers); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDraft(ctx, models.Draft{
		ID: in.DraftID, SenderID: in.UserID, Header: in.Header, Body: in.Body, Receivers: in.Receivers,
	}); err != nil {
		return nil, MapRepositoryError(err)
	}
	saved, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(saved), nil
}

type GetDraftInput struct {
	UserID  int64
	DraftID int64
}

func (s *Service) GetDraftByID(ctx context.Context, in GetDraftInput) (*DraftResult, error) {
	d, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(d), nil
}

type GetDraftsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetDraftsResult struct {
	Drafts []DraftResult
	Limit  int
	Offset int
	Total  int
}

func (s *Service) GetDrafts(ctx context.Context, in GetDraftsInput) (*GetDraftsResult, error) {
	drafts, err := s.repo.GetDrafts(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.CountDraftsByUser(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	out := make([]DraftResult, len(drafts))
	for i := range drafts {
		out[i] = *draftToResult(&drafts[i])
	}
	return &GetDraftsResult{
		Drafts: out,
		Limit:  in.Limit,
		Offset: in.Offset,
		Total:  total,
	}, nil
}

type DeleteDraftInput struct {
	UserID  int64
	DraftID int64
}

func (s *Service) DeleteDraft(ctx context.Context, in DeleteDraftInput) error {
	if err := s.repo.DeleteDraft(ctx, in.DraftID, in.UserID); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

type SendDraftInput struct {
	UserID  int64
	DraftID int64
}

func (s *Service) SendDraft(ctx context.Context, in SendDraftInput) (*SendEmailResult, error) {
	d, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if d.Header == "" || d.Body == "" || len(d.Receivers) == 0 {
		return nil, ErrDraftNotReady
	}
	receiverIDs, err := s.ResolveReceivers(ctx, d.Receivers)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err = s.repo.MarkDraftAsSentTx(ctx, tx, in.DraftID, in.UserID); err != nil {
		return nil, MapRepositoryError(err)
	}
	for _, rid := range receiverIDs {
		if err = s.repo.AddEmailUserWithTx(ctx, tx, in.DraftID, rid, false); err != nil {
			return nil, MapRepositoryError(err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	return &SendEmailResult{
		ID:        in.DraftID,
		SenderID:  in.UserID,
		Header:    d.Header,
		Body:      d.Body,
		CreatedAt: d.CreatedAt,
	}, nil
}

func draftToResult(d *models.Draft) *DraftResult {
	return &DraftResult{
		ID:        d.ID,
		SenderID:  d.SenderID,
		Header:    d.Header,
		Body:      d.Body,
		Receivers: d.Receivers,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func MapRepositoryError(err error) error {
	switch {
	case err == nil:
		return nil
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
	case errors.Is(err, repository.ErrDraftNotFound):
		return ErrEmailNotFound
	case errors.Is(err, repository.ErrAccessDenied):
		return ErrAccessDenied
	default:
		return err
	}
}
