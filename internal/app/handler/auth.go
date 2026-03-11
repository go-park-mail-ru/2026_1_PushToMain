package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

const sessionTokenCookie = "session_token"

type AuthService interface {
	SignUp(ctx context.Context, cmd service.SignUpInput) (string, error)
	SignIn(ctx context.Context, cmd service.SignInInput) (string, error)
}

type SignUpRequest struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

var (
	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._-]+@smail\.ru$`)
	nameRegex    = regexp.MustCompile(`^[a-zA-Zа-яА-Я-]+$`)
	surnameRegex = regexp.MustCompile(`^[a-zA-Zа-яА-Я-]+$`)
)

// @Summary      Регистрация
// @Description  Создаёт нового пользователя и устанавливает сессионную куку
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      handler.SignUpRequest  true  "Данные для регистрации"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  response.ErrorResponse
// @Failure      409    {object}  response.ErrorResponse
// @Failure      500    {object}  response.ErrorResponse
// @Router       /auth/signup [post]
func (handler *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if !ValidateSignUp(req.Email, req.Password, req.Name, req.Surname) {
		response.BadRequest(w)
		return
	}

	token, err := handler.authService.SignUp(r.Context(), service.SignUpInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Surname:  req.Surname,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    token,
		Expires:  time.Now().Add(handler.ttl),
		HttpOnly: true,
		Path:     "/",
	})

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		response.InternalError(w)
		return
	}
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// @Summary      Вход
// @Description  Аутентифицирует пользователя и устанавливает сессионную куку
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      handler.SignInRequest  true  "Данные для входа"
// @Success      200    {object}  map[string]string
// @Failure      400    {object}  response.ErrorResponse
// @Failure      401    {object}  response.ErrorResponse
// @Failure      500    {object}  response.ErrorResponse
// @Router       /auth/signin [post]
func (handler *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if !ValidateSignIn(req.Email, req.Password) {
		response.BadRequest(w)
		return
	}

	token, err := handler.authService.SignIn(r.Context(), service.SignInInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    token,
		Expires:  time.Now().Add(handler.ttl),
		HttpOnly: true,
		Path:     "/",
	})

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		response.InternalError(w)
		return
	}
}

// @Summary      Выход
// @Description  Завершает сессию пользователя, сбрасывает сессионную куку
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  response.ErrorResponse
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {

	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
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

func ValidateSignUp(email, password, name, surname string) bool {

	if email == "" {
		return false
	}

	if !emailRegex.MatchString(email) {
		return false
	}

	if !strings.HasSuffix(email, "@smail.ru") {
		return false
	}

	if len(password) < 8 {
		return false
	}

	if name == "" {
		return false
	}

	if !nameRegex.MatchString(name) {
		return false
	}

	if surname == "" {
		return false
	}

	if !surnameRegex.MatchString(surname) {
		return false
	}

	return true
}

func ValidateSignIn(email, password string) bool {

	if email == "" {
		return false
	}

	if !emailRegex.MatchString(email) {
		return false
	}

	if !strings.HasSuffix(email, "@smail.ru") {
		return false
	}

	if len(password) < 8 {
		return false
	}

	return true
}
