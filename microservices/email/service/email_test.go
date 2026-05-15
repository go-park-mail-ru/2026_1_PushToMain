package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/email"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setup(t *testing.T) (*gomock.Controller, *mocks.MockRepository, *mocks.MockUserClient, *service.Service) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockRepository(ctrl)
	userCli := mocks.NewMockUserClient(ctrl)
	svc := service.New(repo, userCli, service.DraftsConfig{MaxPerUser: 10})
	return ctrl, repo, userCli, svc
}

// ---------- GetEmailsByReceiver ----------
func TestService_GetEmailsByReceiver(t *testing.T) {
	now := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().GetEmailsByReceiver(ctx, int64(1), 10, 0).
			Return([]models.EmailWithMetadata{
				{Email: models.Email{ID: 10, SenderID: 100, Header: "Subj", Body: "Body", CreatedAt: now},
					IsRead: false, IsStarred: true, ReceiversEmails: []string{"r@a.com"}},
			}, nil)
		repo.EXPECT().GetEmailsCount(ctx, int64(1)).Return(5, nil)
		repo.EXPECT().GetUnreadEmailsCount(ctx, int64(1)).Return(2, nil)
		user.EXPECT().GetUserByID(ctx, int64(100)).Return(&userpb.User{Id: 100, Email: "s@a.com", Name: "S", Surname: "K"}, nil)

		res, err := svc.GetEmailsByReceiver(ctx, service.GetEmailsInput{UserID: 1, Limit: 10, Offset: 0})
		require.NoError(t, err)
		require.Len(t, res.Emails, 1)
		assert.Equal(t, int64(10), res.Emails[0].ID)
		assert.Equal(t, "s@a.com", res.Emails[0].SenderEmail)
		assert.Equal(t, 5, res.Total)
		assert.Equal(t, 2, res.UnreadCount)
	})

	t.Run("GetEmailsByReceiver error", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("fail"))
		_, err := svc.GetEmailsByReceiver(ctx, service.GetEmailsInput{UserID: 1})
		assert.Error(t, err)
	})
}

// ---------- GetEmailsBySender ----------
func TestService_GetEmailsBySender(t *testing.T) {
	now := time.Now().UTC()

	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()

		repo.EXPECT().GetEmailsBySender(ctx, int64(1), 5, 0).
			Return([]models.EmailWithMetadata{
				{Email: models.Email{ID: 11, SenderID: 1, Header: "Sent", Body: "Msg", CreatedAt: now},
					IsRead: false, ReceiversEmails: []string{"x"}},
			}, nil)
		repo.EXPECT().GetSenderEmailsCount(ctx, int64(1)).Return(3, nil)

		res, err := svc.GetEmailsBySender(ctx, service.GetMyEmailsInput{UserID: 1, Limit: 5, Offset: 0})
		require.NoError(t, err)
		require.Len(t, res.Emails, 1)
		assert.Equal(t, int64(11), res.Emails[0].ID)
		assert.Equal(t, 3, res.Total)
	})

	t.Run("GetEmailsBySender error", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetEmailsBySender(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("fail"))
		_, err := svc.GetEmailsBySender(ctx, service.GetMyEmailsInput{UserID: 1})
		assert.Error(t, err)
	})
}

// ---------- SendEmail ----------
func TestService_SendEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		in := service.SendEmailInput{
			UserId: 1, Header: "H", Body: "B", Receivers: []string{"x@smail.ru", "y@smail.ru"},
		}

		repo.EXPECT().GetUsersByEmails(ctx, []string{"x@smail.ru", "y@smail.ru"}).
			Return([]*models.User{{ID: 10}, {ID: 20}}, nil)

		db, sqlMock, _ := sqlmock.New()
		defer db.Close()
		sqlMock.ExpectBegin()
		tx, err := db.Begin()
		require.NoError(t, err)

		repo.EXPECT().BeginTx(ctx).Return(tx, nil)

		emailID := int64(100)
		email := models.Email{SenderID: in.UserId, Header: in.Header, Body: in.Body}
		repo.EXPECT().SaveEmailWithTx(ctx, tx, email).Return(emailID, nil)

		repo.EXPECT().AddEmailUserWithTx(ctx, tx, emailID, int64(10), false).Return(nil)
		repo.EXPECT().AddEmailUserWithTx(ctx, tx, emailID, int64(20), false).Return(nil)
		repo.EXPECT().AddEmailUserWithTx(ctx, tx, emailID, int64(1), true).Return(nil)

		sqlMock.ExpectCommit()

		res, err := svc.SendEmail(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, emailID, res.ID)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("resolve receivers error", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUsersByEmails(gomock.Any(), gomock.Any()).Return(nil, errors.New("fail"))
		_, err := svc.SendEmail(ctx, service.SendEmailInput{UserId: 1, Receivers: []string{"x"}})
		assert.Error(t, err)
	})
}

// ---------- ForwardEmail ----------
func TestService_ForwardEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		in := service.ForwardEmailInput{UserID: 2, EmailID: 55, Receivers: []string{"z@smail.ru"}}

		now := time.Now().UTC()
		repo.EXPECT().CheckEmailAccess(ctx, int64(55), int64(2)).Return(nil)
		repo.EXPECT().GetEmailByID(ctx, int64(55)).Return(&models.EmailWithAvatar{
			Email:           models.Email{ID: 55, SenderID: 99, Header: "FwdH", Body: "FwdB", CreatedAt: now},
			SenderImagePath: "/img", ReceiversEmails: []string{"old"},
		}, nil)
		user.EXPECT().GetUserByID(ctx, int64(99)).Return(&userpb.User{Id: 99, Email: "old@smail.ru", Name: "O"}, nil)

		repo.EXPECT().GetUsersByEmails(ctx, []string{"z@smail.ru"}).Return([]*models.User{{ID: 30}}, nil)

		db, sqlMock, _ := sqlmock.New()
		defer db.Close()
		sqlMock.ExpectBegin()
		tx, err := db.Begin()
		require.NoError(t, err)
		repo.EXPECT().BeginTx(ctx).Return(tx, nil)

		newEmailID := int64(200)
		repo.EXPECT().SaveEmailWithTx(ctx, tx, models.Email{SenderID: 2, Header: "FwdH", Body: "FwdB"}).Return(newEmailID, nil)

		repo.EXPECT().AddEmailUserWithTx(ctx, tx, newEmailID, int64(30), false).Return(nil)
		repo.EXPECT().AddEmailUserWithTx(ctx, tx, newEmailID, int64(2), true).Return(nil)
		sqlMock.ExpectCommit()

		err = svc.ForwardEmail(ctx, in)
		require.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("original email not found", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().CheckEmailAccess(ctx, int64(55), int64(2)).Return(nil)
		repo.EXPECT().GetEmailByID(ctx, int64(55)).Return(nil, errors.New("not found"))
		err := svc.ForwardEmail(ctx, service.ForwardEmailInput{UserID: 2, EmailID: 55, Receivers: []string{"x"}})
		assert.Error(t, err)
	})
}

// ---------- GetEmailByID ----------
func TestService_GetEmailByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		in := service.GetEmailInput{UserID: 1, EmailID: 10}
		now := time.Now().UTC()

		repo.EXPECT().CheckEmailAccess(ctx, int64(10), int64(1)).Return(nil)
		repo.EXPECT().GetEmailByID(ctx, int64(10)).Return(&models.EmailWithAvatar{
			Email:           models.Email{ID: 10, SenderID: 99, Header: "H", Body: "B", CreatedAt: now},
			SenderImagePath: "/img", ReceiversEmails: []string{"r1", "r2"},
		}, nil)
		user.EXPECT().GetUserByID(ctx, int64(99)).Return(&userpb.User{Id: 99, Email: "s@smail.ru", Name: "S", Surname: "K"}, nil)

		res, err := svc.GetEmailByID(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, int64(10), res.ID)
		assert.Equal(t, "s@smail.ru", res.SenderEmail)
		assert.Equal(t, []string{"r1", "r2"}, res.ReceiverList)
	})

	t.Run("access denied", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().CheckEmailAccess(ctx, int64(10), int64(1)).Return(errors.New("access denied"))
		_, err := svc.GetEmailByID(ctx, service.GetEmailInput{EmailID: 10, UserID: 1})
		assert.Error(t, err)
	})
}

// ---------- GetEmailsByIDs ----------
func TestService_GetEmailsByIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, user, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		now := time.Now().UTC()

		repo.EXPECT().GetEmailsByIDs(ctx, []int64{10, 20}, int64(1)).Return([]models.EmailWithMetadata{
			{Email: models.Email{ID: 10, SenderID: 100, Header: "A", Body: "B", CreatedAt: now}, IsRead: true, ReceiversEmails: []string{"x"}},
			{Email: models.Email{ID: 20, SenderID: 200, Header: "C", Body: "D", CreatedAt: now}, IsRead: false, ReceiversEmails: []string{"z"}},
		}, nil)

		user.EXPECT().GetUserByID(ctx, int64(100)).Return(&userpb.User{Id: 100, Email: "a@a", Name: "A", Surname: "B"}, nil)
		user.EXPECT().GetUserByID(ctx, int64(200)).Return(&userpb.User{Id: 200, Email: "c@c", Name: "C", Surname: "D"}, nil)

		res, err := svc.GetEmailsByIDs(ctx, []int64{10, 20}, 1)
		require.NoError(t, err)
		require.Len(t, res.Emails, 2)
		assert.Equal(t, 1, res.UnreadCount)
	})

	t.Run("empty IDs", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		res, err := svc.GetEmailsByIDs(ctx, nil, 1)
		require.NoError(t, err)
		assert.Empty(t, res.Emails)
	})
}

// ---------- CheckEmailAccess ----------
func TestService_CheckEmailAccess(t *testing.T) {
	ctrl, repo, _, svc := setup(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo.EXPECT().CheckEmailAccess(ctx, int64(10), int64(1)).Return(nil)
	err := svc.CheckEmailAccess(ctx, service.GetEmailInput{EmailID: 10, UserID: 1})
	assert.NoError(t, err)
}

// ---------- ResolveReceivers ----------
func TestService_ResolveReceivers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().GetUsersByEmails(ctx, []string{"a@a", "b@b"}).Return([]*models.User{{ID: 1}, {ID: 2}}, nil)
		ids, err := svc.ResolveReceivers(ctx, []string{"a@a", "b@b"})
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 2}, ids)
	})

	t.Run("empty input", func(t *testing.T) {
		ctrl, _, _, svc := setup(t)
		defer ctrl.Finish()
		_, err := svc.ResolveReceivers(context.Background(), nil)
		assert.ErrorIs(t, err, service.ErrNoValidReceivers)
	})
}

// ---------- MarkEmailAsRead ----------
func TestService_MarkEmailAsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().MarkEmailAsRead(ctx, int64(10), int64(1)).Return(nil)
		repo.EXPECT().MarkEmailAsRead(ctx, int64(20), int64(1)).Return(nil)
		err := svc.MarkEmailAsRead(ctx, service.MarkAsReadInput{UserID: 1, EmailID: []int64{10, 20}})
		assert.NoError(t, err)
	})
}

// ---------- MarkEmailAsUnRead ----------
func TestService_MarkEmailAsUnRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, repo, _, svc := setup(t)
		defer ctrl.Finish()
		ctx := context.Background()
		repo.EXPECT().MarkEmailAsUnRead(ctx, int64(5), int64(3)).Return(nil)
		err := svc.MarkEmailAsUnRead(ctx, service.MarkAsReadInput{UserID: 3, EmailID: []int64{5}})
		assert.NoError(t, err)
	})
}
