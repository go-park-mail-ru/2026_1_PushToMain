//go:generate mockgen -destination=../../../../mocks/app/user/mock_db_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service DbRepository
//go:generate mockgen -destination=../../../../mocks/app/user/mock_s3_repository.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service S3Repository
//go:generate mockgen -destination=../../../../mocks/app/user/mock_jwt_manager.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service JWTManager

package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/repository/db"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrFailedToGenerateHash = errors.New("failed to generate hash for password")
	ErrFindUser             = errors.New("failed to find user")
	ErrFailedToSaveUser     = errors.New("failed to save user")
	ErrToGenerateJWT        = errors.New("failed to generate jwt")
	ErrWrongPassword        = errors.New("wrong password")
	ErrUploadAvatar         = errors.New("failed to upload avatar")
	ErrUpdateAvatar         = errors.New("failed to update avatar")
	ErrUpdatePassword       = errors.New("failed to update password")
)

type DbRepository interface {
	Save(ctx context.Context, user models.User) (int64, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateAvatar(ctx context.Context, userID int64, imagePath string) error
	FindByID(ctx context.Context, userID int64) (*models.User, error)
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	UpdateProfile(ctx context.Context, userID int64, name, surname string, isMale *bool, birthdate *time.Time) error
}

type S3Repository interface {
	UploadAvatar(ctx context.Context, userID int64, file io.Reader, size int64) (string, error)
	DeleteAvatar(ctx context.Context, userID int64) error
}

type JWTManager interface {
	GenerateJWT(userId int64) (string, error)
	ValidateJWT(token string) (*utils.JwtPayload, error)
}

type Service struct {
	userDB    DbRepository
	s3Storage S3Repository
	jwt       JWTManager
}

func New(r DbRepository, s3 S3Repository, jwt JWTManager) *Service {
	return &Service{
		userDB:    r,
		s3Storage: s3,
		jwt:       jwt,
	}
}

type SignUpInput struct {
	Email    string
	Password string
	Name     string
	Surname  string
}

type UploadAvatarInput struct {
	File   io.Reader
	Size   int64
	UserID int64
}

type UpdatePasswordInput struct {
	UserID      int64
	OldPassword string
	NewPassword string
}

type Folder struct {
	ID   int64
	Name string
}

type GetMeResult struct {
	UserID    int64
	Email     string
	Name      string
	Surname   string
	ImagePath string
	IsMale    *bool
	Birthdate *time.Time
	Folders   []Folder
}

func (s *Service) GetMe(ctx context.Context, userID int64) (*GetMeResult, error) {
	user, err := s.userDB.FindByID(ctx, userID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	folders := make([]Folder, len(user.Folders))
	for i, f := range user.Folders {
		folders[i] = Folder{
			ID:   f.ID,
			Name: f.Name,
		}
	}
	return &GetMeResult{
		UserID:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Surname:   user.Surname,
		ImagePath: user.ImagePath,
		IsMale:    user.IsMale,
		Birthdate: user.Birthdate,
		Folders:   folders,
	}, nil
}

type UpdateProfileInput struct {
	UserID    int64
	Name      string
	Surname   string
	IsMale    *bool
	Birthdate *time.Time
}

func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput) error {
	err := s.userDB.UpdateProfile(ctx, input.UserID, input.Name, input.Surname, input.IsMale, input.Birthdate)
	if err != nil {
		return MapRepositoryError(err)
	}

	return nil
}

func (s *Service) UpdatePassword(ctx context.Context, input UpdatePasswordInput) error {
	user, err := s.userDB.FindByID(ctx, input.UserID)
	if err != nil {
		return MapRepositoryError(err)
	}

	if err := utils.ComparePasswordAndHash(user.Password, input.OldPassword); err != nil {
		return ErrWrongPassword
	}

	hash, err := utils.Hash(input.NewPassword)
	if err != nil {
		return ErrFailedToGenerateHash
	}

	return s.userDB.UpdatePassword(ctx, input.UserID, hash)
}

func (s *Service) UploadAvatar(ctx context.Context, uploadAvatar UploadAvatarInput) (string, error) {
	imagePath, err := s.s3Storage.UploadAvatar(ctx, uploadAvatar.UserID, uploadAvatar.File, uploadAvatar.Size)
	if err != nil {
		return "", ErrUploadAvatar
	}

	err = s.userDB.UpdateAvatar(ctx, uploadAvatar.UserID, imagePath)
	if err != nil {
		if deleteErr := s.s3Storage.DeleteAvatar(ctx, uploadAvatar.UserID); deleteErr != nil {
			return "", deleteErr
		}
		return "", ErrUpdateAvatar
	}

	return imagePath, nil
}

func (s *Service) SignUp(ctx context.Context, signUp SignUpInput) (string, error) {
	_, err := s.userDB.FindByEmail(ctx, signUp.Email)
	if err == nil {
		return "", ErrUserAlreadyExists
	}

	if !errors.Is(err, db.ErrUserNotFound) {
		return "", MapRepositoryError(err)
	}

	hash, err := utils.Hash(signUp.Password)
	if err != nil {
		return "", MapRepositoryError(err)
	}
	userId, err := s.userDB.Save(ctx, models.User{
		Email:    signUp.Email,
		Password: hash,
		Name:     signUp.Name,
		Surname:  signUp.Surname,
	})
	if err != nil {
		return "", MapRepositoryError(err)
	}

	token, err := s.jwt.GenerateJWT(userId)
	if err != nil {
		return "", MapRepositoryError(err)
	}

	return token, nil
}

type SignInInput struct {
	Email    string
	Password string
}

func (s *Service) SignIn(ctx context.Context, signIn SignInInput) (string, error) {
	user, err := s.userDB.FindByEmail(ctx, signIn.Email)
	if err != nil {
		return "", MapRepositoryError(err)
	}

	if err := utils.ComparePasswordAndHash(user.Password, signIn.Password); err != nil {
		return "", MapRepositoryError(err)
	}
	token, err := s.jwt.GenerateJWT(user.ID)
	if err != nil {
		return "", MapRepositoryError(err)
	}

	return token, nil
}

func (s *Service) GenerateToken() (string, error) {
	b := make([]byte, 32)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func MapRepositoryError(err error) error {
	switch {
	case errors.Is(err, db.ErrUserNotFound):
		return ErrUserNotFound
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrWrongPassword
	default:
		return err
	}
}
