//go:generate mockgen -destination=../../../../../mocks/app/email/mock_email_service.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/delivery/http Service

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
	// Spam / Trash листинг
	GetSpamEmails(ctx context.Context, cmd service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetTrashEmails(ctx context.Context, cmd service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetFavoriteEmails(ctx context.Context, cmd service.GetEmailsInput) (*service.GetEmailsResult, error)

	// Массовые действия с письмами
	Trash(ctx context.Context, cmd service.BatchInput) error
	Untrash(ctx context.Context, cmd service.BatchInput) error
	Favorite(ctx context.Context, cmd service.BatchInput) error
	Unfavorite(ctx context.Context, cmd service.BatchInput) error
	Spam(ctx context.Context, cmd service.BatchInput) error
	Unspam(ctx context.Context, cmd service.BatchInput) error
	Delete(ctx context.Context, cmd service.BatchInput) error

	// Drafts
	CreateDraft(ctx context.Context, cmd service.CreateDraftInput) (*service.DraftResult, error)
	UpdateDraft(ctx context.Context, cmd service.UpdateDraftInput) (*service.DraftResult, error)
	GetDraftByID(ctx context.Context, cmd service.GetDraftInput) (*service.DraftResult, error)
	GetDrafts(ctx context.Context, cmd service.GetDraftsInput) (*service.GetDraftsResult, error)
	DeleteDrafts(ctx context.Context, cmd service.DeleteDraftsInput) error
	SendDraft(ctx context.Context, cmd service.SendDraftInput) (*service.SendEmailResult, error)
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

	if len(req.Receivers) == 0 || req.Header == "" || req.Body == "" {
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

	if req.EmailID <= 0 {
		logger.Warn("invalid email_id")
		response.BadRequest(w)
		return
	}
	if payload.UserId <= 0 {
		logger.Warn("invalid user_id")
		response.BadRequest(w)
		return
	}
	if len(req.Receivers) == 0 {
		logger.Warn("empty receivers list")
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
	ID            int64     `json:"id"`
	SenderEmail   string    `json:"sender_email"`
	SenderName    string    `json:"sender_name"`
	SenderSurname string    `json:"sender_surname"`
	ReceiverList  []string  `json:"receiver_list"`
	Header        string    `json:"header"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	IsRead        bool      `json:"is_read"`
	IsFavorite    bool 		`json:"is_favorite"`
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
			ID:            email.ID,
			SenderEmail:   email.SenderEmail,
			SenderName:    email.SenderName,
			SenderSurname: email.SenderSurname,
			ReceiverList:  email.ReceiverList,
			Header:        email.Header,
			Body:          email.Body,
			CreatedAt:     email.CreatedAt,
			IsRead:        email.IsRead,
			IsFavorite:    email.IsStarred,
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
	SenderEmail     string    `json:"sender_email"`
	SenderName      string    `json:"sender_name"`
	SenderSurname   string    `json:"sender_surname"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	SenderImagePath string    `json:"sender_image_path"`
	ReceiverList    []string  `json:"receiver_list"`
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
		SenderEmail:     result.SenderEmail,
		SenderName:      result.SenderName,
		SenderSurname:   result.SenderSurname,
		Header:          result.Header,
		Body:            result.Body,
		CreatedAt:       result.CreatedAt,
		SenderImagePath: result.SenderImagePath,
		ReceiverList:    result.ReceiverList,
	}

	logger.Debugf("Email retrieved successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
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
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as read request received")
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

	emailIDs := []int64{emailID}

	logger.Debugf("Marking email as read, user_id=%d, email_id=%d", payload.UserId, emailID)

	if err := handler.service.MarkEmailAsRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailIDs,
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
// @Router       /api/v1/emails/{id}/unread [put]
func (handler *Handler) MarkEmailAsUnRead(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as unread request received")
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

	emailIDs := []int64{emailID}

	logger.Debugf("Marking email as unread, user_id=%d, email_id=%d", payload.UserId, emailID)

	if err := handler.service.MarkEmailAsUnRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: emailIDs,
	}); err != nil {
		logger.Errorf("Failed to mark email as unread: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email marked as unread successfully: user_id=%d, email_id=%d",
		payload.UserId, emailID)

	w.WriteHeader(http.StatusOK)
}

type MarkEmailsAsReadRequest struct {
	EmailIDs []int64 `json:"email_ids"`
}

// @Summary      Отметить письма как прочитанные
// @Description  Помечает указанное письмо как прочитанное.
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        request body MarkEmailsAsReadRequest true "ID письем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/read [put]
func (handler *Handler) MarkEmailsAsRead(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as read request received")
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

	var req MarkEmailsAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return
	}

	if len(req.EmailIDs) == 0 {
		logger.Warnf("Email IDs array is empty")
		response.BadRequest(w)
		return
	}

	logger.Debugf("Marking emails as read, user_id=%d", payload.UserId)

	if err := handler.service.MarkEmailAsRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: req.EmailIDs,
	}); err != nil {
		logger.Errorf("Failed to mark email as read: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email marked as read successfully: user_id=%d",
		payload.UserId)

	w.WriteHeader(http.StatusOK)
}

// @Summary      Отметить письма как непрочитанные
// @Description  Помечает указанное письмо как непрочитанное.
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        request body MarkEmailsAsReadRequest  true "ID письем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/unread [put]
func (handler *Handler) MarkEmailsAsUnRead(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Mark email as unread request received")
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

	var req MarkEmailsAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return
	}

	if len(req.EmailIDs) == 0 {
		logger.Warnf("Email IDs array is empty")
		response.BadRequest(w)
		return
	}

	logger.Debugf("Marking email as unread, user_id=%d", payload.UserId)

	if err := handler.service.MarkEmailAsUnRead(r.Context(), service.MarkAsReadInput{
		UserID:  payload.UserId,
		EmailID: req.EmailIDs,
	}); err != nil {
		logger.Errorf("Failed to mark email as unread: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Email marked as unread successfully: user_id=%d",
		payload.UserId)

	w.WriteHeader(http.StatusOK)
}
