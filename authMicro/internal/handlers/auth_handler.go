package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/service"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	SignUp(ctx context.Context, cmd service.SignUpInput) (string, error)
	SignIn(ctx context.Context, cmd service.SignInInput) (string, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type AuthResponse struct {
	Token string `json:"token"`
}

type SignUpRequest struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (handler *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	token, err := handler.service.SignUp(r.Context(), service.SignUpInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Surname:  req.Surname,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			response.StatusConflict(w)

		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, AuthResponse{
		Token: token,
	})
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (handler *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	token, err := handler.service.SignIn(r.Context(), service.SignInInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {

		case errors.Is(err, repository.ErrUserNotFound):
			response.Unauthorized(w)

		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			response.Unauthorized(w)

		default:
			response.InternalError(w)
		}

		return
	}

	response.OK(w, AuthResponse{
		Token: token,
	})
}
