package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/folder"
	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestService_CreateNewFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()
	input := service.CreateNewFolderInput{UserId: 1, FolderName: "Work"}

	t.Run("success", func(t *testing.T) {
		repoMock.EXPECT().CountUserFolders(ctx, int64(1)).Return(3, nil)
		repoMock.EXPECT().CreateFolder(ctx, models.Folder{UserID: 1, Name: "Work"}).Return(int64(10), nil)

		result, err := svc.CreateNewFolder(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result.ID)
	})

	t.Run("max folders reached", func(t *testing.T) {
		repoMock.EXPECT().CountUserFolders(ctx, int64(1)).Return(service.MaxCustomFolders, nil)

		result, err := svc.CreateNewFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrMaxFoldersReached)
		assert.Nil(t, result)
	})

	t.Run("CountUserFolders error", func(t *testing.T) {
		repoMock.EXPECT().CountUserFolders(ctx, int64(1)).Return(0, errors.New("db error"))

		result, err := svc.CreateNewFolder(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("duplicate folder", func(t *testing.T) {
		repoMock.EXPECT().CountUserFolders(ctx, int64(1)).Return(2, nil)
		repoMock.EXPECT().CreateFolder(ctx, gomock.Any()).Return(int64(0), repository.ErrDuplicate)

		result, err := svc.CreateNewFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrFolderAlreadyExists)
		assert.Nil(t, result)
	})

	t.Run("CreateFolder db error", func(t *testing.T) {
		repoMock.EXPECT().CountUserFolders(ctx, int64(1)).Return(2, nil)
		repoMock.EXPECT().CreateFolder(ctx, gomock.Any()).Return(int64(0), errors.New("db error"))

		result, err := svc.CreateNewFolder(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestService_ChangeFolderName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()
	input := service.ChangeFolderNameInput{UserID: 1, FolderID: 5, FolderName: "Updated"}

	t.Run("success", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().UpdateFolderName(ctx, int64(5), "Updated").Return(nil)

		err := svc.ChangeFolderName(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("folder not found", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(nil, repository.ErrFolderNotFound)

		err := svc.ChangeFolderName(ctx, input)
		assert.ErrorIs(t, err, service.ErrFolderNotFound)
	})

	t.Run("access denied", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 99}, nil)

		err := svc.ChangeFolderName(ctx, input)
		assert.ErrorIs(t, err, service.ErrAccessDenied)
	})

	t.Run("duplicate name", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().UpdateFolderName(ctx, int64(5), "Updated").Return(repository.ErrDuplicate)

		err := svc.ChangeFolderName(ctx, input)
		assert.ErrorIs(t, err, service.ErrFolderAlreadyExists)
	})
}

func TestService_GetEmailsFromFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()
	input := service.GetEmailsFromFolderInput{UserID: 1, FolderID: 5, Limit: 10, Offset: 0}

	t.Run("success", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().GetFolderEmailIDs(ctx, int64(5), 10, 0).Return([]int64{100, 200}, nil)
		emailMock.EXPECT().GetEmailsByIDs(ctx, []int64{100, 200}, int64(1)).Return(&emailpb.GetEmailsByIdsResponse{
			Emails: []*emailpb.FolderEmail{
				{Id: 100, SenderEmail: "a@smail.ru", SenderName: "A", SenderSurname: "B", Header: "H1", Body: "B1", CreatedAt: timestamppb.Now(), IsRead: false},
				{Id: 200, SenderEmail: "c@smail.ru", SenderName: "C", SenderSurname: "D", Header: "H2", Body: "B2", CreatedAt: timestamppb.Now(), IsRead: true},
			},
			UnreadCount: 1,
		}, nil)
		repoMock.EXPECT().CountEmailsInFolder(ctx, int64(5)).Return(2, nil)

		result, err := svc.GetEmailsFromFolder(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Total)
		assert.Equal(t, 1, result.UnreadCount)
		assert.Len(t, result.Emails, 2)
		assert.Equal(t, "a@smail.ru", result.Emails[0].SenderEmail)
	})

	t.Run("folder not found", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(nil, repository.ErrFolderNotFound)

		result, err := svc.GetEmailsFromFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrFolderNotFound)
		assert.Nil(t, result)
	})

	t.Run("access denied", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 99}, nil)

		result, err := svc.GetEmailsFromFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrAccessDenied)
		assert.Nil(t, result)
	})

	t.Run("GetFolderEmailIDs error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().GetFolderEmailIDs(ctx, int64(5), 10, 0).Return(nil, errors.New("db error"))

		result, err := svc.GetEmailsFromFolder(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("GetEmailsByIDs error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().GetFolderEmailIDs(ctx, int64(5), 10, 0).Return([]int64{100}, nil)
		emailMock.EXPECT().GetEmailsByIDs(ctx, []int64{100}, int64(1)).Return(nil, errors.New("grpc error"))

		result, err := svc.GetEmailsFromFolder(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("CountEmailsInFolder error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().GetFolderEmailIDs(ctx, int64(5), 10, 0).Return([]int64{100}, nil)
		emailMock.EXPECT().GetEmailsByIDs(ctx, gomock.Any(), gomock.Any()).Return(&emailpb.GetEmailsByIdsResponse{
			Emails:      []*emailpb.FolderEmail{},
			UnreadCount: 0,
		}, nil)
		repoMock.EXPECT().CountEmailsInFolder(ctx, int64(5)).Return(0, errors.New("db error"))

		result, err := svc.GetEmailsFromFolder(ctx, input)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestService_AddEmailsInFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		input := service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100, 200}}

		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		emailMock.EXPECT().CheckEmailAccess(ctx, int64(100), int64(1)).Return(true, nil)
		repoMock.EXPECT().AddEmailToFolder(ctx, int64(5), int64(100)).Return(nil)
		emailMock.EXPECT().CheckEmailAccess(ctx, int64(200), int64(1)).Return(true, nil)
		repoMock.EXPECT().AddEmailToFolder(ctx, int64(5), int64(200)).Return(nil)

		err := svc.AddEmailsInFolder(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("empty emails list", func(t *testing.T) {
		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{EmailsID: []int64{}})
		assert.ErrorIs(t, err, service.ErrEmptyEmailsList)
	})

	t.Run("folder not found", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(nil, repository.ErrFolderNotFound)

		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.ErrorIs(t, err, service.ErrFolderNotFound)
	})

	t.Run("access denied", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 99}, nil)

		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.ErrorIs(t, err, service.ErrAccessDenied)
	})

	t.Run("CheckEmailAccess false", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		emailMock.EXPECT().CheckEmailAccess(ctx, int64(100), int64(1)).Return(false, nil)

		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.NoError(t, err)
	})

	t.Run("CheckEmailAccess error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		emailMock.EXPECT().CheckEmailAccess(ctx, int64(100), int64(1)).Return(false, errors.New("grpc error"))

		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.Error(t, err)
	})

	t.Run("AddEmailToFolder error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		emailMock.EXPECT().CheckEmailAccess(ctx, int64(100), int64(1)).Return(true, nil)
		repoMock.EXPECT().AddEmailToFolder(ctx, int64(5), int64(100)).Return(errors.New("db error"))

		err := svc.AddEmailsInFolder(ctx, service.AddEmailsInFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.Error(t, err)
	})
}

func TestService_DeleteEmailsFromFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		input := service.DeleteEmailsFromFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100, 200}}

		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().DeleteEmailFromFolder(ctx, int64(5), int64(100)).Return(nil)
		repoMock.EXPECT().DeleteEmailFromFolder(ctx, int64(5), int64(200)).Return(nil)

		err := svc.DeleteEmailsFromFolder(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("empty emails list", func(t *testing.T) {
		err := svc.DeleteEmailsFromFolder(ctx, service.DeleteEmailsFromFolderInput{EmailsID: []int64{}})
		assert.ErrorIs(t, err, service.ErrEmptyEmailsList)
	})

	t.Run("folder not found", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(nil, repository.ErrFolderNotFound)

		err := svc.DeleteEmailsFromFolder(ctx, service.DeleteEmailsFromFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.ErrorIs(t, err, service.ErrFolderNotFound)
	})

	t.Run("access denied", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 99}, nil)

		err := svc.DeleteEmailsFromFolder(ctx, service.DeleteEmailsFromFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.ErrorIs(t, err, service.ErrAccessDenied)
	})

	t.Run("DeleteEmailFromFolder error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().DeleteEmailFromFolder(ctx, int64(5), int64(100)).Return(errors.New("db error"))

		err := svc.DeleteEmailsFromFolder(ctx, service.DeleteEmailsFromFolderInput{UserID: 1, FolderID: 5, EmailsID: []int64{100}})
		assert.Error(t, err)
	})
}

func TestService_DeleteFolder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMockRepository(ctrl)
	emailMock := mocks.NewMockEmailClient(ctrl)
	svc := service.New(repoMock, emailMock)

	ctx := context.Background()
	input := service.DeleteFolderInput{UserID: 1, FolderID: 5}

	t.Run("success", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().DeleteFolder(ctx, int64(5), int64(1)).Return(nil)

		err := svc.DeleteFolder(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("folder not found", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(nil, repository.ErrFolderNotFound)

		err := svc.DeleteFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrFolderNotFound)
	})

	t.Run("access denied", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 99}, nil)

		err := svc.DeleteFolder(ctx, input)
		assert.ErrorIs(t, err, service.ErrAccessDenied)
	})

	t.Run("DeleteFolder db error", func(t *testing.T) {
		repoMock.EXPECT().GetFolderByID(ctx, int64(5)).Return(&models.Folder{ID: 5, UserID: 1}, nil)
		repoMock.EXPECT().DeleteFolder(ctx, int64(5), int64(1)).Return(errors.New("db error"))

		err := svc.DeleteFolder(ctx, input)
		assert.Error(t, err)
	})
}

func TestMapRepositoryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"duplicate", repository.ErrDuplicate, service.ErrFolderAlreadyExists},
		{"not found", repository.ErrFolderNotFound, service.ErrFolderNotFound},
		{"other error", errors.New("random"), errors.New("random")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.MapRepositoryError(tt.err)
			if tt.want == nil {
				assert.NoError(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
