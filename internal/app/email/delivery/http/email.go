package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type Service interface {
	GetEmailsByReceiver(ctx context.Context, cmd service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetEmailByID(ctx context.Context, cmd service.GetEmailInput) (*service.GetEmailResult, error)
	SendEmail(ctx context.Context, cmd service.SendEmailInput) (*service.SendEmailResult, error)
	ForwardEmail(ctx context.Context, cmd service.ForwardEmailInput) error
	MarkEmailAsRead(ctx context.Context, cmd service.MarkAsReadInput) error
}

type SendEmailRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type SendEmailResponse struct {
	ID        int64     `json:"email_id"`
	SenderID  int64     `json:"from"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// @Summary     Отправить письмо
// @Description  Отправляет письмо получаетлям, которых указал пользователь
// @Tags         emails
// @Produce      json
// @Success      200  {object}   EmailResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/send [post]
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

	if !req.Validate() {
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
	resp := SendEmailResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		CreatedAt: result.CreatedAt,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
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
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/forward [post]
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

	err = handler.service.ForwardEmail(r.Context(), service.ForwardEmailInput{
		UserID:    payload.UserId,
		EmailID:   req.EmailID,
		Receivers: req.Receivers,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type GetEmailsRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type EmailResponse struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	IsRead    bool      `json:"is_read"`
}

type GetEmailsResponse struct {
	Emails []EmailResponse `json:"emails"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Success      200  {array}   EmailResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails [get]
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

	var req GetEmailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	result, err := handler.service.GetEmailsByReceiver(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	emails := make([]EmailResponse, len(result.Emails))
	for i, email := range result.Emails {
		emails[i] = EmailResponse{
			ID:        email.ID,
			SenderID:  email.SenderID,
			Header:    email.Header,
			Body:      email.Body,
			CreatedAt: email.CreatedAt,
			IsRead:    email.IsRead,
		}
	}

	resp := GetEmailsResponse{
		Emails: emails,
		Limit:  result.Limit,
		Offset: result.Offset,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		response.InternalError(w)
		return
	}
}

type GetEmailResponse struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// @Summary      Получить письмо по ID
// @Description  Возвращает детальную информацию о письме
// @Tags         emails
// @Produce      json
// @Param        id   path      int  true  "ID письма"
// @Success      200  {object}  GetEmailResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/{id} [get]
func (handler *Handler) GetEmailByID(w http.ResponseWriter, r *http.Request) {

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		response.BadRequest(w)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[3]

	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w)
		return
	}

	result, err := handler.service.GetEmailByID(r.Context(), service.GetEmailInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}
	resp := GetEmailResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		CreatedAt: result.CreatedAt,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		response.InternalError(w)
		return
	}
}

// @Summary      Отметить письмо как прочитанное
// @Description  Помечает указанное письмо как прочитанное.
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param         id   path      int  true  "ID письма"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/{id}/read [put]
func (handler *Handler) MarkEmailAsRead(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[3]
	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		response.BadRequest(w)
		return
	}

	if err := handler.service.MarkEmailAsRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	}); err != nil {
		parseCommonErrors(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {

	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(w)

	case errors.Is(err, service.ErrEmailNotFound):
		response.NotFound(w)

	case errors.Is(err, service.ErrNoValidReceivers):
		response.NotFound(w)

	case errors.Is(err, service.ErrAccessDenied):
		response.Forbidden(w)

	default:
		response.InternalError(w)
	}
}

func (req *SendEmailRequest) Validate() bool {

	if len(req.Receivers) == 0 {
		return false
	}
	for _, receiver := range req.Receivers {
		_, err := mail.ParseAddressList(receiver)
		if err != nil {
			return false
		}
	}
	return true
}

func (req *ForwardEmailRequest) Validate() bool {
	if len(req.Receivers) == 0 {
		return false
	}
	for _, receiver := range req.Receivers {
		_, err := mail.ParseAddressList(receiver)
		if err != nil {
			return false
		}
	}
	return true
}
