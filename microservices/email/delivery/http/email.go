//go:generate mockgen -destination=../../../../mocks/app/email/mock_email_service.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/http Service

package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
)

// maxSendFormSize limits the entire multipart body when sending an email with attachments.
const maxSendFormSize = 50 << 20 // 50 MB

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
	BlockSenders(ctx context.Context, in service.BatchInput) error
	UnblockSenders(ctx context.Context, in service.BatchInput) error
	Delete(ctx context.Context, in service.BatchInput) error

	CreateDraft(ctx context.Context, in service.CreateDraftInput) (*service.DraftResult, error)
	UpdateDraft(ctx context.Context, in service.UpdateDraftInput) (*service.DraftResult, error)
	GetDraftByID(ctx context.Context, in service.GetDraftInput) (*service.DraftResult, error)
	GetDrafts(ctx context.Context, in service.GetDraftsInput) (*service.GetDraftsResult, error)
	DeleteDrafts(ctx context.Context, in service.DeleteDraftsInput) error
	SendDraft(ctx context.Context, in service.SendDraftInput) (*service.SendEmailResult, error)

	// Attachments
	UploadAttachment(ctx context.Context, in service.UploadAttachmentInput) (*service.AttachmentResult, error)
	DownloadAttachment(ctx context.Context, in service.DownloadAttachmentInput) (*service.DownloadAttachmentResult, error)
	DeleteAttachments(ctx context.Context, in service.DeleteAttachmentsInput) error
	GetAttachments(ctx context.Context, in service.GetAttachmentsInput) (*service.GetAttachmentsResult, error)
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
	resp := GetEmailsResponse{
		Emails:      emails,
		Limit:       result.Limit,
		Offset:      result.Offset,
		Total:       result.Total,
		UnreadCount: result.UnreadCount,
	}
	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		response.InternalError(w)
		return
	}

}

// SendEmail accepts either:
//   - application/json  – plain email without attachments (legacy)
//   - multipart/form-data – email with optional file attachments
//
// For multipart the text fields are: header, body, receivers (JSON array string).
func (h *Handler) SendEmail(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}

	ct := r.Header.Get("Content-Type")
	in := service.SendEmailInput{UserId: userID}

	if isMultipart(ct) {
		// Parse multipart form.
		if err := r.ParseMultipartForm(maxSendFormSize); err != nil {
			response.BadRequest(w)
			return
		}
		in.Header = r.FormValue("header")
		in.Body = r.FormValue("body")

		// receivers is a JSON-encoded string array: '["a@b.com","c@d.com"]'
		receiversRaw := r.FormValue("receivers")
		if receiversRaw != "" {
			if err := json.Unmarshal([]byte(receiversRaw), &in.Receivers); err != nil {
				response.BadRequest(w)
				return
			}
		}

		// Collect uploaded files.
		if r.MultipartForm != nil {
			for _, fhs := range r.MultipartForm.File {
				for _, fh := range fhs {
					f, err := fh.Open()
					if err != nil {
						response.BadRequest(w)
						return
					}
					// Files are closed after SendEmail returns via multipart form GC.
					in.Files = append(in.Files, f)
					in.FileHeaders = append(in.FileHeaders, fh)
				}
			}
		}
	} else {
		// Plain JSON body (no attachments).
		body, err := io.ReadAll(r.Body)
		if err != nil {
			response.BadRequest(w)
			return
		}
		var req SendEmailRequest
		if err := req.UnmarshalJSON(body); err != nil {
			response.BadRequest(w)
			return
		}

		in.Header = req.Header
		in.Body = req.Body
		in.Receivers = req.Receivers
	}

	if (in.Header == "" && in.Body == "") || !validEmails(in.Receivers) {
		response.BadRequest(w)
		return
	}

	result, err := h.service.SendEmail(r.Context(), in)
	if err != nil {
		logger.Errorf("SendEmail: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	resp := SendEmailResponse{
		ID: result.ID, SenderID: result.SenderID,
		Header: result.Header, Body: result.Body, CreatedAt: result.CreatedAt,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (h *Handler) ForwardEmail(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w)
		return
	}
	var req ForwardEmailRequest
	if err := req.UnmarshalJSON(body); err != nil {
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
			ID:              em.ID,
			Header:          em.Header,
			Body:            em.Body,
			CreatedAt:       em.CreatedAt,
			IsRead:          em.IsRead,
			IsStarred:       em.IsStarred,
			ReceiversEmails: em.ReceiversEmails,
		}
	}

	resp := GetMyEmailsResponse{
		Emails: out, Limit: result.Limit, Offset: result.Offset, Total: result.Total,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}

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
	resp := GetEmailResponse{
		ID:              result.ID,
		SenderEmail:     result.SenderEmail,
		SenderName:      result.SenderName,
		SenderSurname:   result.SenderSurname,
		Header:          result.Header,
		Body:            result.Body,
		CreatedAt:       result.CreatedAt,
		SenderImagePath: result.SenderImagePath,
		ReceiverList:    result.ReceiverList,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}

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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w)
		return
	}
	var req MarkEmailsAsReadRequest
	if err := req.UnmarshalJSON(body); err != nil || len(req.EmailIDs) == 0 {
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
