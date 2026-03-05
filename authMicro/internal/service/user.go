package service

import (
	"auth/internal/repository"
	"auth/internal/tools"
	"context"
	"errors"
)

type SignUpCommand struct {
	Email          string
	Password       string
	PasswordRepeat string
	Name           string
	Surname        string
}

type SignInCommand struct {
	Email    string
	Password string
}

type AuthService struct {
	repo repository.IUserRepository
}

var ErrInvalidCredentials = errors.New("invalid password")

func NewAuthService(r repository.IUserRepository) *AuthService {
	return &AuthService{repo: r}
}

func (s *AuthService) SignUp(ctx context.Context, signUp SignUpCommand) (string, error) {
	err := s.ComparePassword(signUp.Password, signUp.PasswordRepeat)
	if err != nil {
		return "", err
	}

	hash, err := tools.Hash(signUp.Password)
	if err != nil {
		return "", err
	}

	user := repository.User{
		Email:    signUp.Email,
		Password: hash,
		Name:     signUp.Name,
		Surname:  signUp.Surname,
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return "", err
	}

	return tools.GenerateJWT(signUp.Email, signUp.Name, signUp.Surname)
}

func (s *AuthService) SignIn(ctx context.Context, signIn SignInCommand) (string, error) {
	user, err := s.repo.FindByEmail(ctx, signIn.Email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := tools.Compare(user.Password, signIn.Password); err != nil {
		return "", ErrInvalidCredentials
	}

	return tools.GenerateJWT(user.Email, user.Name, user.Surname)
}

var ErrPasswordNotEqual = errors.New("passwords do not match")

// валидацию паролей делать здесь или в handler? а потом уже передавать в сервис только 1 пароль
func (s *AuthService) ComparePassword(password, passwordRepeat string) error {
	if password != passwordRepeat {
		return ErrPasswordNotEqual
	}
	return nil
}
