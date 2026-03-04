package service

import (
	"auth/internal/repository"
	"auth/internal/tools"
	"errors"
)

type AuthService struct {
	repo repository.IUserRepository
}

var ErrInvalidCredentials = errors.New("invalid password")

func NewAuthService(r repository.IUserRepository) *AuthService {
	return &AuthService{repo: r}
}

func (s *AuthService) SignUp(email, password, passwordRepeat, name, surname string) (string, error) {
	err := s.ComparePassword(password, passwordRepeat)
	if err != nil {
		return "", err
	}

	hash, err := tools.Hash(password)
	if err != nil {
		return "", err
	}

	user := repository.User{
		Email:    email,
		Password: hash,
		Name:     name,
		Surname:  surname,
	}

	if err := s.repo.Save(user); err != nil {
		return "", err
	}

	return tools.GenerateJWT(email, name, surname)
}

func (s *AuthService) SignIn(email, pass string) (string, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := tools.Compare(user.Password, pass); err != nil {
		return "", ErrInvalidCredentials
	}

	return tools.GenerateJWT(email, user.Name, user.Surname)
}

var ErrPasswordNotEqual = errors.New("passwords do not match")

func (s *AuthService) ComparePassword(password, passwordRepeat string) error {
	if password != passwordRepeat {
		return ErrPasswordNotEqual
	}
	return nil
}
