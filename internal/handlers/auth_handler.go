package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/service"
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
		parseCommonErrors(err, w)
		return
	}

	json.NewEncoder(w).Encode(AuthResponse{
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
		parseCommonErrors(err, w)
		return
	}
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
	})
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {

	case errors.Is(err, service.ErrUserNotFound):
		response.Unauthorized(w)

	case errors.Is(err, service.ErrWrongPassword):
		response.Unauthorized(w)

	case errors.Is(err, service.ErrUserAlreadyExists):
		response.StatusConflict(w)

	default:
		response.InternalError(w)
	}
}
