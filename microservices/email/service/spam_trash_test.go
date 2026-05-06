package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetSpamEmails(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		now := time.Now().UTC()

		repo.EXPECT().GetSpamEmails(ctx, int64(1), 10, 0).
			Return([]models.EmailWithMetadata{
				{Email: models.Email{ID: 99, SenderID: 100, Header: "Spam", Body: "msg", CreatedAt: now},
					IsRead: false, ReceiversEmails: []string{"x"}},
			}, nil)
		repo.EXPECT().GetSpamEmailsCount(ctx, int64(1)).Return(5, nil)
		repo.EXPECT().GetUnreadSpamCount(ctx, int64(1)).Return(3, nil)
		user.EXPECT().GetUserByID(ctx, int64(100)).
			Return(&userpb.User{Id: 100, Email: "s@a.com", Name: "S", Surname: "K"}, nil)

		res, err := svc.GetSpamEmails(ctx, service.GetEmailsInput{UserID: 1, Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 5, res.Total)
		assert.Equal(t, 3, res.UnreadCount)
	})
}

func TestService_GetTrashEmails(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		now := time.Now().UTC()

		repo.EXPECT().GetTrashEmails(ctx, int64(1), 10, 0).
			Return([]models.EmailWithMetadata{
				{Email: models.Email{ID: 77, SenderID: 200, Header: "Trash", Body: "msg", CreatedAt: now},
					IsRead: true, ReceiversEmails: []string{}},
			}, nil)
		repo.EXPECT().GetTrashEmailsCount(ctx, int64(1)).Return(1, nil)
		repo.EXPECT().GetUnreadTrashCount(ctx, int64(1)).Return(0, nil)
		user.EXPECT().GetUserByID(ctx, int64(200)).
			Return(&userpb.User{Id: 200, Email: "t@a.com", Name: "T", Surname: "U"}, nil)

		res, err := svc.GetTrashEmails(ctx, service.GetEmailsInput{UserID: 1, Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 1, res.Total)
	})
}

func TestService_GetFavoriteEmails(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().GetFavoriteEmails(ctx, int64(1), 10, 0).
			Return([]models.EmailWithMetadata{}, nil)

		res, err := svc.GetFavoriteEmails(ctx, service.GetEmailsInput{UserID: 1, Limit: 10, Offset: 0})
		require.NoError(t, err)
		assert.Empty(t, res.Emails)
	})
}
