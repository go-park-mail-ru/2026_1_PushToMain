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

// @Summary Регистрация пользователя
// @Description Создает нового пользователя и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body SignUpRequest true "Данные нового пользователя"
// @Success 200 {object} AuthResponse "JWT токен"
// @Failure 400 {object} map[string]string "BAD_REQUEST"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "User already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /signup [post]
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

	if err := json.NewEncoder(w).Encode(AuthResponse{Token: token}); err != nil {
		response.InternalError(w)
		return
	}
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// @Summary Авторизация пользователя
// @Description Вход пользователя с выдачей JWT токена
// @Tags auth
// @Accept json
// @Produce json
// @Param request body SignInRequest true "Данные для входа"
// @Success 200 {object} AuthResponse "JWT токен"
// @Failure 400 {object} map[string]string "BAD_REQUEST"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /signin [post]
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
	if err := json.NewEncoder(w).Encode(AuthResponse{Token: token}); err != nil {
		response.InternalError(w)
		return
	}
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
