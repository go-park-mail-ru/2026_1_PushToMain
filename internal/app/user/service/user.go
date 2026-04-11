package service

import (
	"context"
	"errors"
	"fmt"
	"io"

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
)

type DbRepository interface {
	Save(ctx context.Context, user models.User) (int64, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateAvatar(ctx context.Context, userID int64, imagePath string) error
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

func (s *Service) UploadAvatar(ctx context.Context, uploadAvatar UploadAvatarInput) (string, error) {
	imagePath, err := s.s3Storage.UploadAvatar(ctx, uploadAvatar.UserID, uploadAvatar.File, uploadAvatar.Size)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUploadAvatar, err)
	}

	err = s.userDB.UpdateAvatar(ctx, uploadAvatar.UserID, imagePath)
	if err != nil {
		if deleteErr := s.s3Storage.DeleteAvatar(ctx, uploadAvatar.UserID); deleteErr != nil {
			return "", fmt.Errorf("update avatar in db: %w; also failed to rollback s3: %v", err, deleteErr)
		}
        return "", fmt.Errorf("update avatar in db: %w", err)
	}

	return imagePath, nil
}

func (s *Service) SignUp(ctx context.Context, signUp SignUpInput) (string, error) {
	_, err := s.userDB.FindByEmail(ctx, signUp.Email)
	if err == nil {
		err = ErrUserAlreadyExists
		return "", fmt.Errorf("faild to signUp bcz user already exist: %w", err)
	}

	if !errors.Is(err, db.ErrUserNotFound) {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	hash, err := utils.Hash(signUp.Password)
	if err != nil {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to generate hash for password: %w", err)
	}
	userId, err := s.userDB.Save(ctx, models.User{
		Email:    signUp.Email,
		Password: hash,
		Name:     signUp.Name,
		Surname:  signUp.Surname,
	})
	if err != nil {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to save user: %w", err)
	}

	token, err := s.jwt.GenerateJWT(userId)
	if err != nil {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to generate jwt: %w", err)
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
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	if err := utils.ComparePasswordAndHash(user.Password, signIn.Password); err != nil {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("wrong password: %w", err)
	}
	token, err := s.jwt.GenerateJWT(user.ID)
	if err != nil {
		err = mapRepositoryError(err)
		return "", fmt.Errorf("failed to generate jwt: %w", err)
	}

	return token, nil
}

func mapRepositoryError(err error) error {
	switch {
	case errors.Is(err, db.ErrUserNotFound):
		return ErrUserNotFound
	case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
		return ErrWrongPassword
	default:
		return err
	}
}
