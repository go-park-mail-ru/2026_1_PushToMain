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
	GetEmailsBySender(ctx context.Context, cmd service.GetMyEmailsInput) (*service.GetMyEmailsResult, error)
	GetEmailByID(ctx context.Context, cmd service.GetEmailInput) (*service.GetEmailResult, error)
	SendEmail(ctx context.Context, cmd service.SendEmailInput) (*service.SendEmailResult, error)
	ForwardEmail(ctx context.Context, cmd service.ForwardEmailInput) error
	MarkEmailAsRead(ctx context.Context, cmd service.MarkAsReadInput) error
	MarkEmailAsUnRead(ctx context.Context, cmd service.MarkAsReadInput) error
	DeleteEmailForReceiver(ctx context.Context, cmd service.DeleteEmailInput) error
	DeleteEmailForSender(ctx context.Context, cmd service.DeleteEmailInput) error
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
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Send email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}

	if !req.Validate() {
		logger.Warnf("Validation failed, user_id=%d: %v", payload.UserId, req.Receivers)
		response.BadRequest(w)
		return
	}

	logger.Warnf("Validation failed, user_id=%d: invalid receivers format", payload.UserId)

	result, err := handler.service.SendEmail(r.Context(), service.SendEmailInput{
		UserId:    payload.UserId,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("Failed to send email: %v", err)
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

	logger.Debugf("Email sent successfully: user_id=%d, email_id=%d",
		payload.UserId, result.ID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
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
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Forward email request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req ForwardEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}
	logger.Debugf("Forwarding email, user_id=%d, email_id=%d, receivers_count=%d",
		payload.UserId, req.EmailID, len(req.Receivers))

	err = handler.service.ForwardEmail(r.Context(), service.ForwardEmailInput{
		UserID:    payload.UserId,
		EmailID:   req.EmailID,
		Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("Failed to forward email: %v", err)
		parseCommonErrors(err, w)
		return
	}
	logger.Debugf("Email forwarded successfully: user_id=%d, email_id=%d",
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
	Emails      []EmailResponse `json:"emails"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
	Total       int             `json:"total"`
	UnreadCount int             `json:"unread_count"`
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
	logger := middleware.GetLogger(r.Context())

	logger.Infof("get email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		logger.Warnf("Invalid user ID: %d", payload.UserId)
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

	logger.Debugf("Getting emails, user_id=%d, limit=%d, offset=%d", payload.UserId, limit, offset)

	result, err := handler.service.GetEmailsByReceiver(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		logger.Errorf("Failed to get emails: %v", err)
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
		Emails:      emails,
		Limit:       result.Limit,
		Offset:      result.Offset,
		Total:       result.Total,
		UnreadCount: result.UnreadCount,
	}

	logger.Debugf("Emails retrieved successfully: user_id=%d, count=%d, total=%d, unread=%d",
		payload.UserId, len(emails), result.Total, result.UnreadCount)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

type MyEmailResponse struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	IsRead          bool      `json:"is_read"`
	ReceiversEmails []string  `json:"receivers_emails"`
}

type GetMyEmailsResponse struct {
	Emails []MyEmailResponse `json:"emails"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

// @Summary      Получить письма отправленные пользователем
// @Description  Возвращает список писем, в которых авторизованный пользователь указан отправителем
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
// @Router       /api/v1/myemails [get]
func (handler *Handler) GetMyEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	logger.Infof("get email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		logger.Warnf("Invalid user ID: %d", payload.UserId)
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

	logger.Debugf("Getting emails, user_id=%d, limit=%d, offset=%d", payload.UserId, limit, offset)

	result, err := handler.service.GetEmailsBySender(r.Context(), service.GetMyEmailsInput{
		UserID: payload.UserId,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		logger.Errorf("Failed to get emails: %v", err)
		parseCommonErrors(err, w)
		return
	}

	emails := make([]MyEmailResponse, len(result.Emails))
	for i, email := range result.Emails {
		emails[i] = MyEmailResponse{
			ID:              email.ID,
			SenderID:        email.SenderID,
			Header:          email.Header,
			Body:            email.Body,
			CreatedAt:       email.CreatedAt,
			IsRead:          email.IsRead,
			ReceiversEmails: email.ReceiversEmails,
		}
	}

	resp := GetMyEmailsResponse{
		Emails: emails,
		Limit:  result.Limit,
		Offset: result.Offset,
		Total:  result.Total,
	}

	logger.Debugf("Emails retrieved successfully: user_id=%d, count=%d, total=%d",
		payload.UserId, len(emails), result.Total)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

type GetEmailResponse struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	SenderImagePath string    `json:"sender_image_path"`
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
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Get email by ID request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		logger.Warnf("Invalid user ID: %d", payload.UserId)
		response.BadRequest(w)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		logger.Warnf("Invalid url %v", err)
		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[4]

	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		logger.Warnf("Invalid email ID format: %s, user_id=%d", emailIDStr, payload.UserId)
		response.BadRequest(w)
		return
	}

	logger.Debugf("Getting email, user_id=%d, email_id=%d", payload.UserId, emailID)

	result, err := handler.service.GetEmailByID(r.Context(), service.GetEmailInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	})
	if err != nil {
		logger.Errorf("Failed to get email: %v", err)
		parseCommonErrors(err, w)
		return
	}
	resp := GetEmailResponse{
		ID:              result.ID,
		SenderID:        result.SenderID,
		Header:          result.Header,
		Body:            result.Body,
		CreatedAt:       result.CreatedAt,
		SenderImagePath: result.SenderImagePath,
	}

	logger.Debugf("Email retrieved successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

type DeleteEmailRequest struct {
	EmailID int64 `json:"email_id"`
}

// @Summary      Удалить письмо (для получателя)
// @Description  Удаляет письмо из почтового ящика получателя (не удаляет само письмо)
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        request body DeleteEmailRequest true "ID письма"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/delete [delete]
func (handler *Handler) DeleteEmailForReceiver(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Delete email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req DeleteEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return
	}

	if req.EmailID <= 0 {
		logger.Warnf("Invalid email ID: %d", req.EmailID)
		response.BadRequest(w)
		return
	}

	logger.Debugf("Deleting email for receiver, user_id=%d, email_id=%d",
		payload.UserId, req.EmailID)

	err = handler.service.DeleteEmailForReceiver(r.Context(), service.DeleteEmailInput{
		UserID:  payload.UserId,
		EmailID: req.EmailID,
	})
	if err != nil {
		logger.Errorf("Failed to delete email: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email deleted successfully, user_id=%d, email_id=%d",
		payload.UserId, req.EmailID)

	w.WriteHeader(http.StatusOK)
}

type DeleteMyEmailRequest struct {
	EmailID int64 `json:"email_id"`
}

// @Summary      Удалить письмо (для отправителя)
// @Description  Удаляет письмо из почтового ящика отправителя (не удаляет само письмо)
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        request body DeleteEmailRequest true "ID письма"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/myemails/delete [delete]
func (handler *Handler) DeleteEmailForSender(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Delete email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req DeleteMyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return
	}

	if req.EmailID <= 0 {
		logger.Warnf("Invalid email ID: %d", req.EmailID)
		response.BadRequest(w)
		return
	}

	logger.Debugf("Deleting email for sender, user_id=%d, email_id=%d",
		payload.UserId, req.EmailID)

	err = handler.service.DeleteEmailForSender(r.Context(), service.DeleteEmailInput{
		UserID:  payload.UserId,
		EmailID: req.EmailID,
	})
	if err != nil {
		logger.Errorf("Failed to delete email: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email deleted successfully, user_id=%d, email_id=%d",
		payload.UserId, req.EmailID)

	w.WriteHeader(http.StatusOK)
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
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as read request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		logger.Warnf("Invalid url %v", err)

		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[4]
	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		logger.Warnf("Invalid email ID format: %s, user_id=%d", emailIDStr, payload.UserId)
		response.BadRequest(w)
		return
	}

	logger.Debugf("Marking email as read, user_id=%d, email_id=%d", payload.UserId, emailID)

	if err := handler.service.MarkEmailAsRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	}); err != nil {
		logger.Errorf("Failed to mark email as read: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email marked as read successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	w.WriteHeader(http.StatusOK)
}

// @Summary      Отметить письмо как непрочитанное
// @Description  Помечает указанное письмо как непрочитанное.
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
func (handler *Handler) MarkEmailAsUnRead(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as unread request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		logger.Warnf("Invalid url %v", err)

		response.BadRequest(w)
		return
	}

	emailIDStr := pathParts[4]
	emailID, err := strconv.ParseInt(emailIDStr, 10, 64)
	if err != nil {
		logger.Warnf("Invalid email ID format: %s, user_id=%d", emailIDStr, payload.UserId)
		response.BadRequest(w)
		return
	}

	logger.Debugf("Marking email as unread, user_id=%d, email_id=%d", payload.UserId, emailID)

	if err := handler.service.MarkEmailAsUnRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	}); err != nil {
		logger.Errorf("Failed to mark email as unread: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email marked as unread successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	w.WriteHeader(http.StatusOK)
}
