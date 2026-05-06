package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/user"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer_GetUserById(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	srv := New(mockSvc)

	birth := time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC)
	male := true
	userID := int64(42)

	t.Run("success", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), userID).
			Return(&service.GetMeResult{
				UserID:    userID,
				Email:     "john@smail.ru",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatars/42.jpg",
				IsMale:    &male,
				Birthdate: &birth,
				Folders: []service.Folder{
					{ID: 1, Name: "inbox"},
				},
			}, nil)

		resp, err := srv.GetUserById(context.Background(), &userpb.GetUserByIdRequest{
			UserId: userID,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, userID, resp.User.Id)
		assert.Equal(t, "john@smail.ru", resp.User.Email)
		assert.Equal(t, "John", resp.User.Name)
		assert.Equal(t, "Doe", resp.User.Surname)
		assert.Equal(t, "/avatars/42.jpg", resp.User.ImagePath)
		assert.True(t, resp.User.IsMale)
		assert.Equal(t, "2000-01-15 00:00:00 +0000 UTC", resp.User.Birthdate)
	})

	t.Run("user not found", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), int64(99)).
			Return(nil, service.ErrUserNotFound)

		resp, err := srv.GetUserById(context.Background(), &userpb.GetUserByIdRequest{
			UserId: 99,
		})

		require.Error(t, err)
		assert.Nil(t, resp)

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("nil IsMale and Birthdate", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), userID).
			Return(&service.GetMeResult{
				UserID:    userID,
				Email:     "jane@smail.ru",
				Name:      "Jane",
				Surname:   "Doe",
				ImagePath: "",
				IsMale:    nil,
				Birthdate: nil,
				Folders:   []service.Folder{},
			}, nil)

		resp, err := srv.GetUserById(context.Background(), &userpb.GetUserByIdRequest{
			UserId: userID,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.User.IsMale)
		assert.Empty(t, resp.User.Birthdate)
	})
}

func TestServer_UserExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	srv := New(mockSvc)

	t.Run("user exists", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), int64(1)).
			Return(&service.GetMeResult{UserID: 1}, nil)

		resp, err := srv.UserExists(context.Background(), &userpb.UserExistsRequest{
			UserId: 1,
		})

		require.NoError(t, err)
		assert.True(t, resp.Exists)
	})

	t.Run("user does not exist", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), int64(99)).
			Return(nil, service.ErrUserNotFound)

		resp, err := srv.UserExists(context.Background(), &userpb.UserExistsRequest{
			UserId: 99,
		})

		require.NoError(t, err)
		assert.False(t, resp.Exists)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc.EXPECT().
			GetMe(gomock.Any(), int64(1)).
			Return(nil, errors.New("db error"))

		resp, err := srv.UserExists(context.Background(), &userpb.UserExistsRequest{
			UserId: 1,
		})

		require.NoError(t, err)
		assert.False(t, resp.Exists)
	})
}
