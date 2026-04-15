package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/repository/db"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestService_GetMe(t *testing.T) {
	connRefusedErr := errors.New("connection refused")
	tests := []struct {
		name          string
		userID        int64
		setupMock     func(*mocks.MockDbRepository)
		expected      *GetMeResult
		expectedError error
	}{
		{
			name:   "success",
			userID: 123,
			setupMock: func(m *mocks.MockDbRepository) {
				m.EXPECT().
					FindByID(gomock.Any(), int64(123)).
					Return(&models.User{
						ID:        123,
						Email:     "user@example.com",
						Name:      "John",
						Surname:   "Doe",
						ImagePath: "/avatars/123.jpg",
					}, nil)
			},
			expected: &GetMeResult{
				UserID:    123,
				Email:     "user@example.com",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatars/123.jpg",
			},
			expectedError: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMock: func(m *mocks.MockDbRepository) {
				m.EXPECT().
					FindByID(gomock.Any(), int64(999)).
					Return(nil, db.ErrUserNotFound)
			},
			expected:      nil,
			expectedError: ErrUserNotFound,
		},
		{
			name:   "database error",
			userID: 123,
			setupMock: func(m *mocks.MockDbRepository) {
				m.EXPECT().
					FindByID(gomock.Any(), int64(123)).
					Return(nil, connRefusedErr)
			},
			expected:      nil,
			expectedError: connRefusedErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			tt.setupMock(mockDB)

			s := New(mockDB, nil, nil)
			result, err := s.GetMe(context.Background(), tt.userID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestService_UpdatePassword(t *testing.T) {
	dbErr := errors.New("database error")
	tests := []struct {
		name          string
		input         UpdatePasswordInput
		setupMock     func(*mocks.MockDbRepository)
		expectedError error
	}{
		{
			name: "success",
			input: UpdatePasswordInput{
				UserID:      123,
				OldPassword: "correct-old",
				NewPassword: "new-secure-password",
			},
			setupMock: func(m *mocks.MockDbRepository) {
				// Mock FindByID
				hashedOld, _ := utils.Hash("correct-old")
				m.EXPECT().
					FindByID(gomock.Any(), int64(123)).
					Return(&models.User{
						ID:       123,
						Password: hashedOld,
					}, nil)
				// Mock UpdatePassword
				m.EXPECT().
					UpdatePassword(gomock.Any(), int64(123), gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			input: UpdatePasswordInput{
				UserID:      999,
				OldPassword: "old",
				NewPassword: "new",
			},
			setupMock: func(m *mocks.MockDbRepository) {
				m.EXPECT().
					FindByID(gomock.Any(), int64(999)).
					Return(nil, db.ErrUserNotFound)
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedError: ErrUserNotFound,
		},
		{
			name: "wrong old password",
			input: UpdatePasswordInput{
				UserID:      123,
				OldPassword: "wrong-old",
				NewPassword: "new",
			},
			setupMock: func(m *mocks.MockDbRepository) {
				hashedCorrect, _ := utils.Hash("correct-old")
				m.EXPECT().
					FindByID(gomock.Any(), int64(123)).
					Return(&models.User{
						ID:       123,
						Password: hashedCorrect,
					}, nil)
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			expectedError: ErrWrongPassword,
		},
		{
			name: "update password DB error",
			input: UpdatePasswordInput{
				UserID:      123,
				OldPassword: "correct-old",
				NewPassword: "new",
			},
			setupMock: func(m *mocks.MockDbRepository) {
				hashedCorrect, _ := utils.Hash("correct-old")
				m.EXPECT().
					FindByID(gomock.Any(), int64(123)).
					Return(&models.User{
						ID:       123,
						Password: hashedCorrect,
					}, nil)
				m.EXPECT().
					UpdatePassword(gomock.Any(), int64(123), gomock.Any()).
					Return(dbErr)
			},
			expectedError: dbErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			tt.setupMock(mockDB)

			s := New(mockDB, nil, nil)
			err := s.UpdatePassword(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_UploadAvatar(t *testing.T) {
	tests := []struct {
		name          string
		input         UploadAvatarInput
		setupMock     func(*mocks.MockDbRepository, *mocks.MockS3Repository)
		expectedPath  string
		expectedError error
	}{
		{
			name: "success",
			input: UploadAvatarInput{
				UserID: 123,
				File:   strings.NewReader("image data"),
				Size:   1024,
			},
			setupMock: func(db *mocks.MockDbRepository, s3 *mocks.MockS3Repository) {
				s3.EXPECT().
					UploadAvatar(gomock.Any(), int64(123), gomock.Any(), int64(1024)).
					Return("/avatars/123.jpg", nil)
				db.EXPECT().
					UpdateAvatar(gomock.Any(), int64(123), "/avatars/123.jpg").
					Return(nil)
			},
			expectedPath:  "/avatars/123.jpg",
			expectedError: nil,
		},
		{
			name: "S3 upload fails",
			input: UploadAvatarInput{
				UserID: 123,
				File:   strings.NewReader("image data"),
				Size:   1024,
			},
			setupMock: func(db *mocks.MockDbRepository, s3 *mocks.MockS3Repository) {
				s3.EXPECT().
					UploadAvatar(gomock.Any(), int64(123), gomock.Any(), int64(1024)).
					Return("", errors.New("s3 error"))
				db.EXPECT().UpdateAvatar(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				s3.EXPECT().DeleteAvatar(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedPath:  "",
			expectedError: ErrUploadAvatar,
		},
		{
			name: "DB update fails, S3 rollback succeeds",
			input: UploadAvatarInput{
				UserID: 123,
				File:   strings.NewReader("image data"),
				Size:   1024,
			},
			setupMock: func(db *mocks.MockDbRepository, s3 *mocks.MockS3Repository) {
				s3.EXPECT().
					UploadAvatar(gomock.Any(), int64(123), gomock.Any(), int64(1024)).
					Return("/avatars/123.jpg", nil)
				db.EXPECT().
					UpdateAvatar(gomock.Any(), int64(123), "/avatars/123.jpg").
					Return(errors.New("db error"))
				s3.EXPECT().
					DeleteAvatar(gomock.Any(), int64(123)).
					Return(nil)
			},
			expectedPath:  "",
			expectedError: ErrUpdateAvatar,
		},
		{
			name: "DB update fails, S3 rollback also fails",
			input: UploadAvatarInput{
				UserID: 123,
				File:   strings.NewReader("image data"),
				Size:   1024,
			},
			setupMock: func(db *mocks.MockDbRepository, s3 *mocks.MockS3Repository) {
				s3.EXPECT().
					UploadAvatar(gomock.Any(), int64(123), gomock.Any(), int64(1024)).
					Return("/avatars/123.jpg", nil)
				db.EXPECT().
					UpdateAvatar(gomock.Any(), int64(123), "/avatars/123.jpg").
					Return(errors.New("db error"))
				s3.EXPECT().
					DeleteAvatar(gomock.Any(), int64(123)).
					Return(errors.New("s3 delete error"))
			},
			expectedPath:  "",
			expectedError: errors.New("s3 delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			mockS3 := mocks.NewMockS3Repository(ctrl)
			tt.setupMock(mockDB, mockS3)

			s := New(mockDB, mockS3, nil)
			path, err := s.UploadAvatar(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError) || err.Error() == tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

func TestService_SignUp(t *testing.T) {
	tests := []struct {
		name          string
		input         SignUpInput
		setupMock     func(*mocks.MockDbRepository, *mocks.MockJWTManager)
		expectedToken string
		expectedError error
	}{
		{
			name: "success",
			input: SignUpInput{
				Email:    "new@example.com",
				Password: "password123",
				Name:     "Alice",
				Surname:  "Smith",
			},
			setupMock: func(dbRepo *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				dbRepo.EXPECT().
					FindByEmail(gomock.Any(), "new@example.com").
					Return(nil, db.ErrUserNotFound)
				dbRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(int64(456), nil)
				jwt.EXPECT().
					GenerateJWT(int64(456)).
					Return("valid-jwt-token", nil)
			},
			expectedToken: "valid-jwt-token",
			expectedError: nil,
		},
		{
			name: "user already exists",
			input: SignUpInput{
				Email:    "existing@example.com",
				Password: "pass",
			},
			setupMock: func(dbRepo *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				dbRepo.EXPECT().
					FindByEmail(gomock.Any(), "existing@example.com").
					Return(&models.User{ID: 1}, nil)
				dbRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)
				jwt.EXPECT().GenerateJWT(gomock.Any()).Times(0)
			},
			expectedToken: "",
			expectedError: ErrUserAlreadyExists,
		},
		{
			name: "FindByEmail unexpected error",
			input: SignUpInput{
				Email: "error@example.com",
			},
			setupMock: func(dbRepo *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				dbRepo.EXPECT().
					FindByEmail(gomock.Any(), "error@example.com").
					Return(nil, errors.New("db connection lost"))
				dbRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedToken: "",
			expectedError: errors.New("db connection lost"),
		},
		{
			name: "Save fails",
			input: SignUpInput{
				Email:    "new@example.com",
				Password: "pass",
			},
			setupMock: func(dbRepo *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				dbRepo.EXPECT().
					FindByEmail(gomock.Any(), "new@example.com").
					Return(nil, db.ErrUserNotFound)
				dbRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("duplicate key"))
				jwt.EXPECT().GenerateJWT(gomock.Any()).Times(0)
			},
			expectedToken: "",
			expectedError: errors.New("duplicate key"),
		},
		{
			name: "JWT generation fails",
			input: SignUpInput{
				Email:    "new@example.com",
				Password: "pass",
			},
			setupMock: func(dbRepo *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				dbRepo.EXPECT().
					FindByEmail(gomock.Any(), "new@example.com").
					Return(nil, db.ErrUserNotFound)
				dbRepo.EXPECT().
					Save(gomock.Any(), gomock.Any()).
					Return(int64(456), nil)
				jwt.EXPECT().
					GenerateJWT(int64(456)).
					Return("", errors.New("jwt signing error"))
			},
			expectedToken: "",
			expectedError: errors.New("jwt signing error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			mockJWT := mocks.NewMockJWTManager(ctrl)
			tt.setupMock(mockDB, mockJWT)

			s := New(mockDB, nil, mockJWT)
			token, err := s.SignUp(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError) || strings.Contains(err.Error(), tt.expectedError.Error()))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestService_SignIn(t *testing.T) {
	tests := []struct {
		name          string
		input         SignInInput
		setupMock     func(*mocks.MockDbRepository, *mocks.MockJWTManager)
		expectedToken string
		expectedError error
	}{
		{
			name: "success",
			input: SignInInput{
				Email:    "user@example.com",
				Password: "correct-password",
			},
			setupMock: func(db *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				hashed, _ := utils.Hash("correct-password")
				db.EXPECT().
					FindByEmail(gomock.Any(), "user@example.com").
					Return(&models.User{
						ID:       123,
						Password: hashed,
					}, nil)
				jwt.EXPECT().
					GenerateJWT(int64(123)).
					Return("valid-jwt", nil)
			},
			expectedToken: "valid-jwt",
			expectedError: nil,
		},
		{
			name: "user not found",
			input: SignInInput{
				Email:    "missing@example.com",
				Password: "any",
			},
			setupMock: func(db *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				db.EXPECT().
					FindByEmail(gomock.Any(), "missing@example.com").
					Return(nil, ErrUserNotFound)
				jwt.EXPECT().GenerateJWT(gomock.Any()).Times(0)
			},
			expectedToken: "",
			expectedError: ErrUserNotFound,
		},
		{
			name: "wrong password",
			input: SignInInput{
				Email:    "user@example.com",
				Password: "wrong",
			},
			setupMock: func(db *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				hashedCorrect, _ := utils.Hash("correct-password")
				db.EXPECT().
					FindByEmail(gomock.Any(), "user@example.com").
					Return(&models.User{
						ID:       123,
						Password: hashedCorrect,
					}, nil)
				jwt.EXPECT().GenerateJWT(gomock.Any()).Times(0)
			},
			expectedToken: "",
			expectedError: ErrWrongPassword,
		},
		{
			name: "JWT generation fails",
			input: SignInInput{
				Email:    "user@example.com",
				Password: "correct-password",
			},
			setupMock: func(db *mocks.MockDbRepository, jwt *mocks.MockJWTManager) {
				hashed, _ := utils.Hash("correct-password")
				db.EXPECT().
					FindByEmail(gomock.Any(), "user@example.com").
					Return(&models.User{
						ID:       123,
						Password: hashed,
					}, nil)
				jwt.EXPECT().
					GenerateJWT(int64(123)).
					Return("", errors.New("jwt error"))
			},
			expectedToken: "",
			expectedError: errors.New("jwt error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			mockJWT := mocks.NewMockJWTManager(ctrl)
			tt.setupMock(mockDB, mockJWT)

			s := New(mockDB, nil, mockJWT)
			token, err := s.SignIn(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError) || strings.Contains(err.Error(), tt.expectedError.Error()))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestService_GenerateToken(t *testing.T) {
	s := New(nil, nil, nil)

	token, err := s.GenerateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	assert.Len(t, token, 44)
}

func Test_mapRepositoryError(t *testing.T) {
	unknownErr := errors.New("unknown error")
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{"db.ErrUserNotFound -> ErrUserNotFound", db.ErrUserNotFound, ErrUserNotFound},
		{"bcrypt.ErrMismatchedHashAndPassword -> ErrWrongPassword", bcrypt.ErrMismatchedHashAndPassword, ErrWrongPassword},
		{"unknown error unchanged", unknownErr, unknownErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRepositoryError(tt.err)
			assert.True(t, errors.Is(result, tt.expected))
		})
	}
}

func TestService_UpdateProfile(t *testing.T) {
	tests := []struct {
		name          string
		input         UpdateProfileInput
		setupMock     func(*mocks.MockDbRepository)
		expectedError error
	}{
		{
			name: "success",
			input: UpdateProfileInput{
				UserID:  123,
				Name:    "John",
				Surname: "Doe",
			},
			setupMock: func(db *mocks.MockDbRepository) {
				db.EXPECT().
					UpdateProfile(gomock.Any(), int64(123), "John", "Doe").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			input: UpdateProfileInput{
				UserID:  123,
				Name:    "John",
				Surname: "Doe",
			},
			setupMock: func(db *mocks.MockDbRepository) {
				db.EXPECT().
					UpdateProfile(gomock.Any(), int64(123), "John", "Doe").
					Return(ErrUserNotFound)
			},
			expectedError: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mocks.NewMockDbRepository(ctrl)
			tt.setupMock(mockDB)

			s := New(mockDB, nil, nil)
			err := s.UpdateProfile(context.Background(), tt.input)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
