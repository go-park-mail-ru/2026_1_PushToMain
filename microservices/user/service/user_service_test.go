package service_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/repository/db"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return string(hash)
}

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbRepo := mocks.NewMockDbRepository(ctrl)
	s3Repo := mocks.NewMockS3Repository(ctrl)
	jwtMgr := mocks.NewMockJWTManager(ctrl)

	svc := service.New(dbRepo, s3Repo, jwtMgr)
	assert.NotNil(t, svc)
}

func TestService_GetMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	userID := int64(1)

	t.Run("success", func(t *testing.T) {
		u := &models.User{
			ID:        userID,
			Email:     "a@b.com",
			Name:      "John",
			Surname:   "Doe",
			ImagePath: "/img.jpg",
			IsMale:    boolPtr(true),
			Birthdate: timePtr(time.Date(1999, 1, 2, 0, 0, 0, 0, time.UTC)),
			Folders: []models.Folder{
				{ID: 10, Name: "inbox"},
				{ID: 20, Name: "sent"},
			},
		}
		dbMock.EXPECT().FindByID(ctx, userID).Return(u, nil)

		res, err := svc.GetMe(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, &service.GetMeResult{
			UserID:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			Surname:   u.Surname,
			ImagePath: u.ImagePath,
			IsMale:    u.IsMale,
			Birthdate: u.Birthdate,
			Folders: []service.Folder{
				{ID: 10, Name: "inbox"},
				{ID: 20, Name: "sent"},
			},
		}, res)
	})

	t.Run("user not found", func(t *testing.T) {
		dbMock.EXPECT().FindByID(ctx, userID).Return(nil, db.ErrUserNotFound)

		res, err := svc.GetMe(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("db error", func(t *testing.T) {
		someErr := errors.New("connection lost")
		dbMock.EXPECT().FindByID(ctx, userID).Return(nil, someErr)

		res, err := svc.GetMe(ctx, userID)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, someErr)
	})

	t.Run("empty folders", func(t *testing.T) {
		u := &models.User{
			ID:      userID,
			Folders: []models.Folder{},
		}
		dbMock.EXPECT().FindByID(ctx, userID).Return(u, nil)

		res, err := svc.GetMe(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, []service.Folder{}, res.Folders)
	})
}

func TestService_UpdateProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	input := service.UpdateProfileInput{
		UserID:  1,
		Name:    "Jane",
		Surname: "Roe",
		IsMale:  boolPtr(false),
	}

	t.Run("success", func(t *testing.T) {
		dbMock.EXPECT().UpdateProfile(ctx, input.UserID, input.Name, input.Surname, input.IsMale, gomock.Any()).Return(nil)
		err := svc.UpdateProfile(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		dbMock.EXPECT().UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(db.ErrUserNotFound)
		err := svc.UpdateProfile(ctx, input)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("db error", func(t *testing.T) {
		dbMock.EXPECT().UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db down"))
		err := svc.UpdateProfile(ctx, input)
		assert.ErrorContains(t, err, "db down")
	})
}

func TestService_UpdatePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	oldPassword := "oldSecret"
	newPassword := "newSecret"
	oldHash := hashPassword(t, oldPassword)

	user := &models.User{
		ID:       1,
		Password: oldHash,
	}

	input := service.UpdatePasswordInput{
		UserID:      user.ID,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}

	t.Run("success", func(t *testing.T) {
		dbMock.EXPECT().FindByID(ctx, user.ID).Return(user, nil)
		dbMock.EXPECT().UpdatePassword(ctx, user.ID, gomock.Any()).Return(nil)
		err := svc.UpdatePassword(ctx, input)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		dbMock.EXPECT().FindByID(ctx, int64(999)).Return(nil, db.ErrUserNotFound)
		err := svc.UpdatePassword(ctx, service.UpdatePasswordInput{
			UserID:      999,
			OldPassword: "x",
			NewPassword: "y",
		})
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("wrong old password", func(t *testing.T) {
		dbMock.EXPECT().FindByID(ctx, user.ID).Return(user, nil)
		err := svc.UpdatePassword(ctx, service.UpdatePasswordInput{
			UserID:      user.ID,
			OldPassword: "wrong",
			NewPassword: newPassword,
		})
		assert.ErrorIs(t, err, service.ErrWrongPassword)
	})

	t.Run("update fails", func(t *testing.T) {
		dbMock.EXPECT().FindByID(ctx, user.ID).Return(user, nil)
		dbMock.EXPECT().UpdatePassword(ctx, user.ID, gomock.Any()).Return(db.ErrUserNotFound)
		err := svc.UpdatePassword(ctx, input)
		assert.ErrorIs(t, err, db.ErrUserNotFound)
	})
}

func TestService_UploadAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	userID := int64(42)
	fileContent := "fake image data"
	file := strings.NewReader(fileContent)
	size := int64(len(fileContent))
	imagePath := "users/42/avatar"

	t.Run("success", func(t *testing.T) {
		s3Mock.EXPECT().UploadAvatar(ctx, userID, gomock.Any(), size).Return(imagePath, nil)
		dbMock.EXPECT().UpdateAvatar(ctx, userID, imagePath).Return(nil)
		result, err := svc.UploadAvatar(ctx, service.UploadAvatarInput{
			UserID: userID,
			File:   file,
			Size:   size,
		})
		require.NoError(t, err)
		assert.Equal(t, imagePath, result)
	})

	t.Run("s3 upload fails", func(t *testing.T) {
		s3Mock.EXPECT().UploadAvatar(ctx, userID, gomock.Any(), size).Return("", errors.New("s3 error"))
		_, err := svc.UploadAvatar(ctx, service.UploadAvatarInput{
			UserID: userID,
			File:   file,
			Size:   size,
		})
		assert.ErrorIs(t, err, service.ErrUploadAvatar)
	})

	t.Run("db update fails, s3 delete succeeds", func(t *testing.T) {
		s3Mock.EXPECT().UploadAvatar(ctx, userID, gomock.Any(), size).Return(imagePath, nil)
		dbMock.EXPECT().UpdateAvatar(ctx, userID, imagePath).Return(errors.New("db failed"))
		s3Mock.EXPECT().DeleteAvatar(ctx, userID).Return(nil)
		_, err := svc.UploadAvatar(ctx, service.UploadAvatarInput{
			UserID: userID,
			File:   file,
			Size:   size,
		})
		assert.ErrorIs(t, err, service.ErrUpdateAvatar)
	})

	t.Run("db update fails, s3 delete also fails", func(t *testing.T) {
		s3Mock.EXPECT().UploadAvatar(ctx, userID, gomock.Any(), size).Return(imagePath, nil)
		dbMock.EXPECT().UpdateAvatar(ctx, userID, imagePath).Return(errors.New("db failed"))
		s3DeleteErr := errors.New("s3 delete failed")
		s3Mock.EXPECT().DeleteAvatar(ctx, userID).Return(s3DeleteErr)
		_, err := svc.UploadAvatar(ctx, service.UploadAvatarInput{
			UserID: userID,
			File:   file,
			Size:   size,
		})
		assert.ErrorIs(t, err, s3DeleteErr)
	})
}

func TestService_SignUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	email := "new@user.com"
	password := "secret123"
	name := "Alice"
	surname := "Bob"
	token := "jwt.token.here"

	input := service.SignUpInput{
		Email:    email,
		Password: password,
		Name:     name,
		Surname:  surname,
	}

	t.Run("success", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(nil, db.ErrUserNotFound)
		dbMock.EXPECT().Save(ctx, gomock.Any()).Return(int64(1), nil)
		jwtMock.EXPECT().GenerateJWT(int64(1)).Return(token, nil)
		result, err := svc.SignUp(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, token, result)
	})

	t.Run("user already exists", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(&models.User{ID: 99}, nil)
		_, err := svc.SignUp(ctx, input)
		assert.ErrorIs(t, err, service.ErrUserAlreadyExists)
	})

	t.Run("FindByEmail unexpected error", func(t *testing.T) {
		dbErr := errors.New("db timeout")
		dbMock.EXPECT().FindByEmail(ctx, email).Return(nil, dbErr)
		_, err := svc.SignUp(ctx, input)
		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("Save fails", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(nil, db.ErrUserNotFound)
		dbMock.EXPECT().Save(ctx, gomock.Any()).Return(int64(0), db.ErrQueryError)
		_, err := svc.SignUp(ctx, input)
		assert.ErrorIs(t, err, db.ErrQueryError)
	})

	t.Run("GenerateJWT fails", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(nil, db.ErrUserNotFound)
		dbMock.EXPECT().Save(ctx, gomock.Any()).Return(int64(2), nil)
		jwtErr := errors.New("jwt engine down")
		jwtMock.EXPECT().GenerateJWT(int64(2)).Return("", jwtErr)
		_, err := svc.SignUp(ctx, input)
		assert.ErrorIs(t, err, jwtErr)
	})
}

func TestService_SignIn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mocks.NewMockDbRepository(ctrl)
	s3Mock := mocks.NewMockS3Repository(ctrl)
	jwtMock := mocks.NewMockJWTManager(ctrl)
	svc := service.New(dbMock, s3Mock, jwtMock)

	ctx := context.Background()
	email := "existing@user.com"
	plainPassword := "correct"
	wrongPassword := "wrong"
	token := "jwt.token.here"

	hashedPassword := hashPassword(t, plainPassword)
	user := &models.User{
		ID:       5,
		Email:    email,
		Password: hashedPassword,
	}

	input := service.SignInInput{
		Email:    email,
		Password: plainPassword,
	}

	t.Run("success", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(user, nil)
		jwtMock.EXPECT().GenerateJWT(user.ID).Return(token, nil)
		result, err := svc.SignIn(ctx, input)
		require.NoError(t, err)
		assert.Equal(t, token, result)
	})

	t.Run("user not found", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(nil, db.ErrUserNotFound)
		_, err := svc.SignIn(ctx, input)
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("wrong password", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(user, nil)
		_, err := svc.SignIn(ctx, service.SignInInput{Email: email, Password: wrongPassword})
		assert.ErrorIs(t, err, service.ErrWrongPassword)
	})

	t.Run("jwt generation fails", func(t *testing.T) {
		dbMock.EXPECT().FindByEmail(ctx, email).Return(user, nil)
		jwtMock.EXPECT().GenerateJWT(user.ID).Return("", errors.New("jwt error"))
		_, err := svc.SignIn(ctx, input)
		assert.ErrorContains(t, err, "jwt error")
	})
}

func TestService_GenerateToken(t *testing.T) {
	svc := &service.Service{}
	token, err := svc.GenerateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	decoded, err := base64.StdEncoding.DecodeString(token)
	assert.NoError(t, err)
	assert.Equal(t, 32, len(decoded))
	assert.Len(t, token, 44)
}

func TestMapRepositoryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"user not found", db.ErrUserNotFound, service.ErrUserNotFound},
		{"bcrypt mismatch", bcrypt.ErrMismatchedHashAndPassword, service.ErrWrongPassword},
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

func boolPtr(b bool) *bool           { return &b }
func timePtr(t time.Time) *time.Time { return &t }
