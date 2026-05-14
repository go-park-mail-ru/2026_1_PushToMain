//go:generate mockgen -destination=../../../../mocks/app/email/mock_email_service.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/http Service

package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
)

type Service interface {
	GetAllEmailsByUser(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetEmailsByReceiver(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetEmailsBySender(ctx context.Context, in service.GetMyEmailsInput) (*service.GetMyEmailsResult, error)
	GetEmailByID(ctx context.Context, in service.GetEmailInput) (*service.GetEmailResult, error)
	SendEmail(ctx context.Context, in service.SendEmailInput) (*service.SendEmailResult, error)
	ForwardEmail(ctx context.Context, in service.ForwardEmailInput) error
	MarkEmailAsRead(ctx context.Context, in service.MarkAsReadInput) error
	MarkEmailAsUnRead(ctx context.Context, in service.MarkAsReadInput) error

	GetSpamEmails(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetTrashEmails(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error)
	GetFavoriteEmails(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error)

	Trash(ctx context.Context, in service.BatchInput) error
	Untrash(ctx context.Context, in service.BatchInput) error
	Favorite(ctx context.Context, in service.BatchInput) error
	Unfavorite(ctx context.Context, in service.BatchInput) error
	Spam(ctx context.Context, in service.BatchInput) error
	Unspam(ctx context.Context, in service.BatchInput) error
	UnmarkSpamSenders(ctx context.Context, in service.BatchInput) error
	Delete(ctx context.Context, in service.BatchInput) error

	CreateDraft(ctx context.Context, in service.CreateDraftInput) (*service.DraftResult, error)
	UpdateDraft(ctx context.Context, in service.UpdateDraftInput) (*service.DraftResult, error)
	GetDraftByID(ctx context.Context, in service.GetDraftInput) (*service.DraftResult, error)
	GetDrafts(ctx context.Context, in service.GetDraftsInput) (*service.GetDraftsResult, error)
	DeleteDrafts(ctx context.Context, in service.DeleteDraftsInput) error
	SendDraft(ctx context.Context, in service.SendDraftInput) (*service.SendEmailResult, error)
}

func emailToDTO(em service.EmailResult) EmailResponse {
	return EmailResponse{
		ID:            em.ID,
		SenderEmail:   em.SenderEmail,
		SenderName:    em.SenderName,
		SenderSurname: em.SenderSurname,
		ReceiverList:  em.ReceiverList,
		Header:        em.Header,
		Body:          em.Body,
		CreatedAt:     em.CreatedAt,
		IsRead:        em.IsRead,
		IsStarred:     em.IsStarred,
	}
}

func writeEmailsList(w http.ResponseWriter, result *service.GetEmailsResult) {
	emails := make([]EmailResponse, len(result.Emails))
	for i, em := range result.Emails {
		emails[i] = emailToDTO(em)
	}
	writeJSON(w, http.StatusOK, GetEmailsResponse{
		Emails:      emails,
		Limit:       result.Limit,
		Offset:      result.Offset,
		Total:       result.Total,
		UnreadCount: result.UnreadCount,
	})
}

func (h *Handler) SendEmail(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	var req SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}
	if req.Header == "" || req.Body == "" || !validEmails(req.Receivers) {
		response.BadRequest(w)
		return
	}
	result, err := h.service.SendEmail(r.Context(), service.SendEmailInput{
		UserId: userID, Header: req.Header, Body: req.Body, Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("SendEmail: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	writeJSON(w, http.StatusOK, SendEmailResponse{
		ID: result.ID, SenderID: result.SenderID,
		Header: result.Header, Body: result.Body, CreatedAt: result.CreatedAt,
	})
}

func (h *Handler) ForwardEmail(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	var req ForwardEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}
	if req.EmailID <= 0 || !validEmails(req.Receivers) {
		response.BadRequest(w)
		return
	}
	if err := h.service.ForwardEmail(r.Context(), service.ForwardEmailInput{
		UserID: userID, EmailID: req.EmailID, Receivers: req.Receivers,
	}); err != nil {
		logger.Errorf("ForwardEmail: user_id=%d, email_id=%d, err=%v", userID, req.EmailID, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetInboxEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetEmailsByReceiver(r.Context(), service.GetEmailsInput{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetInbox: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, result)
}

func (h *Handler) GetAllEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetAllEmailsByUser(r.Context(), service.GetEmailsInput{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetAllEmails: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, result)
}

func (h *Handler) GetSentEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetEmailsBySender(r.Context(), service.GetMyEmailsInput{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetSent: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	out := make([]MyEmailResponse, len(result.Emails))
	for i, em := range result.Emails {
		out[i] = MyEmailResponse{
			ID: em.ID, SenderID: em.SenderID,
			Header: em.Header, Body: em.Body,
			CreatedAt: em.CreatedAt, IsRead: em.IsRead, IsStarred: em.IsStarred,
			ReceiversEmails: em.ReceiversEmails,
		}
	}
	writeJSON(w, http.StatusOK, GetMyEmailsResponse{
		Emails: out, Limit: result.Limit, Offset: result.Offset, Total: result.Total,
	})
}

func (h *Handler) GetEmailByID(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}
	result, err := h.service.GetEmailByID(r.Context(), service.GetEmailInput{
		UserID: userID, EmailID: emailID,
	})
	if err != nil {
		logger.Errorf("GetEmailByID: user_id=%d, email_id=%d, err=%v", userID, emailID, err)
		parseCommonErrors(err, w)
		return
	}
	writeJSON(w, http.StatusOK, GetEmailResponse{
		ID: result.ID, SenderID: result.SenderID,
		SenderEmail: result.SenderEmail, SenderName: result.SenderName, SenderSurname: result.SenderSurname,
		Header: result.Header, Body: result.Body,
		CreatedAt: result.CreatedAt, SenderImagePath: result.SenderImagePath,
		ReceiverList: result.ReceiverList,
	})
}

func (h *Handler) GetSpamEmails(w http.ResponseWriter, r *http.Request) {
	h.getFolder(w, r, "GetSpam", h.service.GetSpamEmails)
}

func (h *Handler) GetTrashEmails(w http.ResponseWriter, r *http.Request) {
	h.getFolder(w, r, "GetTrash", h.service.GetTrashEmails)
}

func (h *Handler) GetFavoriteEmails(w http.ResponseWriter, r *http.Request) {
	h.getFolder(w, r, "GetFavorites", h.service.GetFavoriteEmails)
}

func (h *Handler) getFolder(
	w http.ResponseWriter, r *http.Request, op string,
	fn func(ctx context.Context, in service.GetEmailsInput) (*service.GetEmailsResult, error),
) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	result, err := fn(r.Context(), service.GetEmailsInput{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("%s: user_id=%d, err=%v", op, userID, err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, result)
}

func (h *Handler) MarkEmailAsRead(w http.ResponseWriter, r *http.Request) {
	h.markReadSingle(w, r, true)
}

func (h *Handler) MarkEmailAsUnRead(w http.ResponseWriter, r *http.Request) {
	h.markReadSingle(w, r, false)
}

func (h *Handler) MarkEmailsAsRead(w http.ResponseWriter, r *http.Request) {
	h.markReadBatch(w, r, true)
}

func (h *Handler) MarkEmailsAsUnRead(w http.ResponseWriter, r *http.Request) {
	h.markReadBatch(w, r, false)
}

func (h *Handler) markReadSingle(w http.ResponseWriter, r *http.Request, isRead bool) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}
	in := service.MarkAsReadInput{UserID: userID, EmailID: []int64{emailID}}
	if err := h.callMarkRead(r.Context(), in, isRead); err != nil {
		logger.Errorf("MarkRead: user_id=%d, email_id=%d, isRead=%t, err=%v", userID, emailID, isRead, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) markReadBatch(w http.ResponseWriter, r *http.Request, isRead bool) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	var req MarkEmailsAsReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.EmailIDs) == 0 {
		response.BadRequest(w)
		return
	}
	for _, id := range req.EmailIDs {
		if id <= 0 {
			response.BadRequest(w)
			return
		}
	}
	in := service.MarkAsReadInput{UserID: userID, EmailID: req.EmailIDs}
	if err := h.callMarkRead(r.Context(), in, isRead); err != nil {
		logger.Errorf("MarkReadBatch: user_id=%d, isRead=%t, err=%v", userID, isRead, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) callMarkRead(ctx context.Context, in service.MarkAsReadInput, isRead bool) error {
	if isRead {
		return h.service.MarkEmailAsRead(ctx, in)
	}
	return h.service.MarkEmailAsUnRead(ctx, in)
}
