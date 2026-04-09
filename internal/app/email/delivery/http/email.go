package handler

import (
	"context"
	"encoding/json"
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

	requestID := r.Header.Get("X-Request-ID")
	handler.Logger.Infof("Send email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		handler.Logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.Logger.Warnw("Invalid request body",
			"request_id", requestID,
			"user_id", payload.UserId,
			"error", err,
		)
		response.BadRequest(w)
		return
	}

	if !req.Validate() {
		handler.Logger.Warnw("Validation failed",
			"request_id", requestID,
			"user_id", payload.UserId,
			"receivers", req.Receivers,
		)
		response.BadRequest(w)
		return
	}

	handler.Logger.Debugw("Sending email",
		"request_id", requestID,
		"user_id", payload.UserId,
		"receivers_count", len(req.Receivers),
		"header", req.Header,
	)

	result, err := handler.service.SendEmail(r.Context(), service.SendEmailInput{
		UserId:    payload.UserId,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receivers,
	})
	if err != nil {
		handler.Logger.Errorf("Failed to send email: %v", err)
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

	handler.Logger.Debugf("Email sent successfully: user_id=%d, email_id=%d",
		payload.UserId, result.ID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		handler.Logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
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
	requestID := r.Header.Get("X-Request-ID")
	handler.Logger.Infof("Forward email request received",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
	)
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		handler.Logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req ForwardEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.Logger.Warnw("Invalid request body",
			"request_id", requestID,
			"user_id", payload.UserId,
			"error", err,
		)
		response.BadRequest(w)
		return
	}
	handler.Logger.Debugw("Forwarding email",
		"request_id", requestID,
		"user_id", payload.UserId,
		"email_id", req.EmailID,
		"receivers_count", len(req.Receivers),
	)
	err = handler.service.ForwardEmail(r.Context(), service.ForwardEmailInput{
		UserID:    payload.UserId,
		EmailID:   req.EmailID,
		Receivers: req.Receivers,
	})
	if err != nil {
		handler.Logger.Errorf("Failed to forward email: %v", err)
		parseCommonErrors(err, w)
		return
	}
	handler.Logger.Debugf("Email forwarded successfully: user_id=%d, email_id=%d",
		payload.UserId, req.EmailID)

	w.WriteHeader(http.StatusOK)
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
	Total  int             `json:"total"`
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Количество записей на странице (default: 20, max: 100)"
// @Param        offset  query     int  false  "Смещение для пагинации (default: 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails [get]
func (handler *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {

	handler.Logger.Infof("Send email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		handler.Logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		handler.Logger.Warnw("Invalid user ID",
			"user_id", payload.UserId,
		)
		response.BadRequest(w)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	handler.Logger.Debugw("Getting emails with pagination",
		"user_id", payload.UserId,
		"limit", limit,
		"offset", offset,
	)

	result, err := handler.service.GetEmailsByReceiver(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		handler.Logger.Errorf("Failed to get emails: %v", err)
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
		Total:  result.Total,
	}

	handler.Logger.Debugf("Emails retrieved successfully: user_id=%d, count=%d, total=%d",
		payload.UserId, len(emails), result.Total)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		handler.Logger.Errorf("Failed to encode response: %v", err)
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
	handler.Logger.Infof("Get email by ID request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		handler.Logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		handler.Logger.Warnw("Invalid user ID",
			"user_id", payload.UserId,
		)
		response.BadRequest(w)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		handler.Logger.Warnw("Invalid url %v", err)
		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[4]

	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		handler.Logger.Warnw("Invalid email ID",
			"email_id", emailIDStr,
			"error", err,
		)
		response.BadRequest(w)
		return
	}

	handler.Logger.Debugw("Getting email by ID",
		"user_id", payload.UserId,
		"email_id", emailID,
	)

	result, err := handler.service.GetEmailByID(r.Context(), service.GetEmailInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	})
	if err != nil {
		handler.Logger.Errorf("Failed to get email: %v", err)
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

	handler.Logger.Debugf("Email retrieved successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		handler.Logger.Errorf("Failed to encode response: %v", err)
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
	handler.Logger.Infof("Mark email as read request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		handler.Logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		handler.Logger.Warnf("Invalid url %v", err)

		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[4]
	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		handler.Logger.Warnw("Invalid email ID",
			"email_id", emailIDStr,
			"error", err,
		)
		response.BadRequest(w)
		return
	}

	handler.Logger.Debugw("Marking email as read",
		"user_id", payload.UserId,
		"email_id", emailID,
	)

	if err := handler.service.MarkEmailAsRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	}); err != nil {
		handler.Logger.Errorf("Failed to mark email as read: %v", err)
		parseCommonErrors(err, w)
		return
	}

	handler.Logger.Debugf("Email marked as read successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	w.WriteHeader(http.StatusOK)
}
