package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/tools"
)

var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepository interface {
	Save(ctx context.Context, user models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type AuthService struct {
	repo UserRepository
}

func NewAuthService(r UserRepository) *AuthService {
	return &AuthService{repo: r}
}

type SignUpInput struct {
	Email    string
	Password string
	Name     string
	Surname  string
}

func (s *AuthService) SignUp(ctx context.Context, signUp SignUpInput) (string, error) {
	_, err := s.repo.FindByEmail(ctx, signUp.Email)

	if err == nil {
		return "", fmt.Errorf("faild to signUp bcz user already exist: %w", ErrUserAlreadyExists)
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	hash, err := tools.Hash(signUp.Password)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash for password: %w", err)
	}

	if err := s.repo.Save(ctx, models.User{
		Email:    signUp.Email,
		Password: hash,
		Name:     signUp.Name,
		Surname:  signUp.Surname,
	}); err != nil {
		return "", fmt.Errorf("failed to save user: %w", err)
	}

	token, err := tools.GenerateJWT(signUp.Email, signUp.Name, signUp.Surname)
	if err != nil {
		return "", fmt.Errorf("failed to generate jwt: %w", err)
	}

	return token, nil
}

type SignInInput struct {
	Email    string
	Password string
}

func (s *AuthService) SignIn(ctx context.Context, signIn SignInInput) (string, error) {
	user, err := s.repo.FindByEmail(ctx, signIn.Email)
	if err != nil {
		return "", fmt.Errorf("failed to find user by email: %w", err)
	}

	if err := tools.ComparePasswordAndHash(user.Password, signIn.Password); err != nil {
		return "", fmt.Errorf("failed to compare passwords: %w", err)
	}
	token, err := tools.GenerateJWT(user.Email, user.Name, user.Surname)
	if err != nil {
		return "", fmt.Errorf("failed to generate jwt: %w", err)
	}

	return token, nil
}
