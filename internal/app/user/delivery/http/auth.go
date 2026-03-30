//go:generate mockgen -destination=../mocks/mock_service.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/http Service

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

const sessionTokenCookie = "session_token"

type Service interface {
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
// @Router       api/v1/auth/signup [post]
func (handler *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if !req.Validate() {
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

	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    token,
		Expires:  time.Now().Add(handler.cfg.TTL),
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
// @Router       api/v1/auth/signin [post]
func (handler *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if !req.Validate() {
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

	http.SetCookie(w, &http.Cookie{
		Name:     sessionTokenCookie,
		Value:    token,
		Expires:  time.Now().Add(handler.cfg.TTL),
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
// @Router       api/v1/auth/logout [post]
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

func (req *SignUpRequest) Validate() bool {

	if req.Email == "" {
		return false
	}

	if !emailRegex.MatchString(req.Email) {
		return false
	}

	if !strings.HasSuffix(req.Email, "@smail.ru") {
		return false
	}

	if len(req.Password) < 8 {
		return false
	}

	if req.Name == "" {
		return false
	}

	if !nameRegex.MatchString(req.Name) {
		return false
	}

	if req.Surname == "" {
		return false
	}

	if !surnameRegex.MatchString(req.Surname) {
		return false
	}

	return true
}

func (req *SignInRequest) Validate() bool {

	if req.Email == "" {
		return false
	}

	if !emailRegex.MatchString(req.Email) {
		return false
	}

	if !strings.HasSuffix(req.Email, "@smail.ru") {
		return false
	}

	if len(req.Password) < 8 {
		return false
	}

	return true
}
