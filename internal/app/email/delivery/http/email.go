package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type Service interface {
	GetEmailsByReceiver(ctx context.Context, userId int64) ([]models.Email, error)
	SendEmail(ctx context.Context, cmd service.SendEmailInput) (*models.Email, error)
}

type SendEmailRequest struct {
	Header   string `json:"header"`
	Body     string `json:"body"`
	Receiver string `json:"receiver"`
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

	result, err := handler.service.SendEmail(r.Context(), service.SendEmailInput{
		UserId:    payload.UserId,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receiver,
	})
	if err != nil {
		response.InternalError(w)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
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
		response.InternalError(w)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		response.InternalError(w)
		return
	}
}
