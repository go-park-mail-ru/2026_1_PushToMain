package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type Service interface {
	GetEmailsByReceiver(ctx context.Context, userId int64) ([]models.Email, error)
	SendEmail(ctx context.Context, cmd service.SendEmailInput) (*models.Email, error)
	ForwardEmail(ctx context.Context, cmd service.ForwardEmailInput) error
}

type SendEmailRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

// @Summary     Отправить письмо
// @Description  Отправляет письмо получаетлям, которых указал пользователь
// @Tags         emails
// @Produce      json
// @Success      200  {object}   models.Email
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       api/v1/send [post]
func (handler *Handler) SendEmail(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if payload.UserId <= 0 {
		response.BadRequest(w)
		return
	}

	if len(req.Receivers) == 0 {
		response.BadRequest(w)
		return
	}

	result, err := handler.service.SendEmail(r.Context(), service.SendEmailInput{
		UserId:    payload.UserId,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receivers,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		response.InternalError(w)
		return
	}
}

type ForwardEmailRequest struct {
	EmailID   int64    `json:"email_id"`
	Receivers []string `json:"receivers"`
}

// @Summary     Переслать письмо
// @Description  Пересылает письмо получаетлям, которых указал пользователь
// @Tags         emails
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       api/v1/forward [post]
func (handler *Handler) ForwardEmail(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	var req ForwardEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	if payload.UserId <= 0 {
		response.BadRequest(w)
		return
	}

	if len(req.Receivers) == 0 {
		response.BadRequest(w)
		return
	}

	err = handler.service.ForwardEmail(r.Context(), service.ForwardEmailInput{
		UserID:    payload.UserId,
		EmailID:   req.EmailID,
		Receivers: req.Receivers,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		response.InternalError(w)
		return
	}
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Success      200  {array}   models.Email
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       api/v1/emails [get]
func (handler *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		response.BadRequest(w)
		return
	}

	result, err := handler.service.GetEmailsByReceiver(r.Context(), payload.UserId)
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		response.InternalError(w)
		return
	}
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {

	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(w)

	case errors.Is(err, service.ErrNoValidReceivers):
		response.NotFound(w)

	case errors.Is(err, service.ErrAccessDenied):
		response.Forbidden(w)

	default:
		response.InternalError(w)
	}
}
