//go:generate mockgen -destination=../../../../mocks/app/email/mock_email_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service Repository

package service

import (
	"context"
	"database/sql"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
)

type UserClient interface {
	GetUserByID(
		ctx context.Context,
		userID int64,
	) (*userpb.User, error)

	UserExists(
		ctx context.Context,
		userID int64,
	) (bool, error)
}

type Repository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	SaveEmail(ctx context.Context, email models.Email) (int64, error)
	SaveEmailWithTx(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error)
	AddEmailUserWithTx(ctx context.Context, tx *sql.Tx, emailID, userID int64, isSender bool) error
	GetUsersByEmails(ctx context.Context, emails []string) ([]*models.User, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error)
	GetUserEmailFlags(ctx context.Context, emailID, userID int64, isSender bool) (*models.UserEmail, error)
	CheckUserEmailExists(ctx context.Context, emailID, userID int64) (bool, error)
	CheckEmailAccess(ctx context.Context, emailID, userID int64) error

	GetEmailsByReceiver(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetEmailsBySender(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadEmailsCount(ctx context.Context, userID int64) (int, error)
	GetSenderEmailsCount(ctx context.Context, userID int64) (int, error)
	MarkEmailAsRead(ctx context.Context, emailID, userID int64) error
	MarkEmailAsUnRead(ctx context.Context, emailID, userID int64) error

	GetSpamEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetSpamEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadSpamCount(ctx context.Context, userID int64) (int, error)
	GetTrashEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetTrashEmailsCount(ctx context.Context, userID int64) (int, error)
	GetUnreadTrashCount(ctx context.Context, userID int64) (int, error)
	GetFavoriteEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)

	SetStarredBatch(ctx context.Context, userID int64, emailIDs []int64, starred bool) error
	SetTrashedBatch(ctx context.Context, userID int64, emailIDs []int64, trashed bool) error
	SetSpamBatch(ctx context.Context, userID int64, emailIDs []int64, spam bool) error
	MarkSendersAsSpamBatch(ctx context.Context, userID int64, emailIDs []int64) error
	UnmarkSendersAsSpamBatch(ctx context.Context, userID int64, emailIDs []int64) error
	HardDeleteBatch(ctx context.Context, userID int64, emailIDs []int64) error

	// Drafts
	CountDraftsByUser(ctx context.Context, userID int64) (int, error)
	CreateDraft(ctx context.Context, draft models.Draft) (int64, error)
	UpdateDraft(ctx context.Context, draft models.Draft) error
	GetDraftByID(ctx context.Context, draftID, userID int64) (*models.Draft, error)
	GetDrafts(ctx context.Context, userID int64, limit, offset int) ([]models.Draft, error)
	DeleteDraftsBatch(ctx context.Context, userID int64, draftIDs []int64) error
	MarkDraftAsSentTx(ctx context.Context, tx *sql.Tx, draftID, userID int64) error
}

type DraftsConfig struct {
	MaxPerUser int
}

type Service struct {
	repo       Repository
	drafts     DraftsConfig
	userClient UserClient
}

func New(repo Repository, userClient UserClient, drafts DraftsConfig) *Service {
	return &Service{
		repo:       repo,
		drafts:     drafts,
		userClient: userClient,
	}
}
