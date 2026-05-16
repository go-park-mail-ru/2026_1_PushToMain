package service

import (
	"context"
	"database/sql"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
)

//go:generate mockgen -destination=../../../mocks/app/email/mock_email_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service Repository
//go:generate mockgen -destination=../../../mocks/app/email/mock_email_user_client.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service UserClient

type UserClient interface {
	GetUserByID(ctx context.Context, userID int64) (*userpb.User, error)
	UserExists(ctx context.Context, userID int64) (bool, error)
	GetUsersByEmails(ctx context.Context, emails []string) ([]*userpb.User, error)
}

type Repository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)

	InsertEmail(ctx context.Context, tx *sql.Tx, email models.Email) (int64, error)
	InsertEmailRecipients(ctx context.Context, tx *sql.Tx, emailID int64, recipients []models.Recipient) error
	InsertUserEmail(ctx context.Context, tx *sql.Tx, userID, emailID int64, checkSpam bool) error

	GetAllEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetEmailByID(ctx context.Context, emailID int64) (*models.EmailWithAvatar, error)
	GetInboxEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetSentEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetSpamEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetTrashEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)
	GetFavoriteEmails(ctx context.Context, userID int64, limit, offset int) ([]models.EmailWithMetadata, error)

	GetInboxStats(ctx context.Context, userID int64) (models.MailboxStats, error)
	GetSpamStats(ctx context.Context, userID int64) (models.MailboxStats, error)
	GetTrashStats(ctx context.Context, userID int64) (models.MailboxStats, error)
	GetFavoritesStats(ctx context.Context, userID int64) (models.MailboxStats, error)
	CountSentEmails(ctx context.Context, userID int64) (int, error)
	CountDrafts(ctx context.Context, userID int64) (int, error)

	CheckEmailAccess(ctx context.Context, userID, emailID int64) error
	GetDeletedEmailIDs(ctx context.Context, userID int64, emailIDs []int64) ([]int64, error)

	GetSpamEmailIDs(ctx context.Context, userID int64, emailIDs []int64) ([]int64, error)
	StarEmails(ctx context.Context, userID int64, emailIDs []int64) error
	UnstarEmails(ctx context.Context, userID int64, emailIDs []int64) error
	SpamEmails(ctx context.Context, userID int64, emailIDs []int64) error
	UnspamEmails(ctx context.Context, userID int64, emailIDs []int64) error
	TrashEmails(ctx context.Context, userID int64, emailIDs []int64) error
	UntrashEmails(ctx context.Context, userID int64, emailIDs []int64) error
	ReadEmails(ctx context.Context, userID int64, emailIDs []int64) error
	UnreadEmails(ctx context.Context, userID int64, emailIDs []int64) error
	DeleteUserEmailsBatch(ctx context.Context, userID int64, emailIDs []int64) error

	UnblockSendersBatch(ctx context.Context, userID int64, senderIDs []int64) error
	BlockSendersBatch(ctx context.Context, userID int64, senderIDs []int64) error
	CreateDraft(ctx context.Context, draft models.Draft) (*models.Draft, error)
	UpdateDraft(ctx context.Context, userID int64, draft models.Draft) error
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
	return &Service{repo: repo, drafts: drafts, userClient: userClient}
}
