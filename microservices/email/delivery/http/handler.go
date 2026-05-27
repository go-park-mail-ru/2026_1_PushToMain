package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/gorilla/mux"
)

type Config struct {
	TTL time.Duration
}

type Handler struct {
	service Service
	cfg     Config
}

func New(service Service, cfg Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	private.HandleFunc("/all-emails", h.GetAllEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/inbox", h.GetInboxEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/sent", h.GetSentEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/send", h.SendEmail).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/forward", h.ForwardEmail).Methods(http.MethodPost, http.MethodOptions)

	private.HandleFunc("/emails/spam", h.GetSpamEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/spam", h.Spam).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/unspam", h.Unspam).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/trash", h.GetTrashEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/trash", h.Trash).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/untrash", h.Untrash).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/favorite", h.GetFavoriteEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/favorite", h.Favorite).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/unfavorite", h.Unfavorite).Methods(http.MethodPut, http.MethodOptions)

	private.HandleFunc("/emails/read", h.MarkEmailsAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/unread", h.MarkEmailsAsUnRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/read", h.MarkEmailAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/unread", h.MarkEmailAsUnRead).Methods(http.MethodPut, http.MethodOptions)

	private.HandleFunc("/emails", h.Delete).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/spam-senders", h.UnmarkSpamSenders).Methods(http.MethodDelete, http.MethodOptions)

	// Вложения: специфичный маршрут /{attachment_id} регистрируется раньше общего /attachments,
	// чтобы gorilla/mux не перехватил его раньше.
	private.HandleFunc("/emails/{id:[0-9]+}/attachments/{attachment_id:[0-9]+}", h.DownloadAttachment).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.GetAttachments).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.UploadAttachment).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.DeleteAttachments).Methods(http.MethodDelete, http.MethodOptions)

	private.HandleFunc("/emails/{id:[0-9]+}", h.GetEmailByID).Methods(http.MethodGet, http.MethodOptions)

	// Attachment routes — note: specific path /attachments/{attachment_id} must be
	// registered BEFORE the general /attachments to avoid route shadowing.
	private.HandleFunc("/emails/{id:[0-9]+}/attachments/{attachment_id:[0-9]+}", h.DownloadAttachment).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.GetAttachments).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.UploadAttachment).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/attachments", h.DeleteAttachments).Methods(http.MethodDelete, http.MethodOptions)

	private.HandleFunc("/drafts", h.CreateDraft).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/drafts", h.GetDrafts).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/drafts", h.DeleteDrafts).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/drafts/{id:[0-9]+}", h.GetDraftByID).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/drafts/{id:[0-9]+}", h.UpdateDraft).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/drafts/{id:[0-9]+}/send", h.SendDraft).Methods(http.MethodPost, http.MethodOptions)
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	// Сначала проверяем типизированные ошибки через errors.As,
	// потому что errors.Is не работает со структурами (каждый раз новый экземпляр).
	var errSavedAsDraft *service.ErrSavedAsDraft
	if errors.As(err, &errSavedAsDraft) {
		// 202 Accepted: запрос принят, но письмо не отправлено — сохранено как черновик.
		// Фронтенд должен уведомить пользователя и предложить повторить отправку из черновиков.
		writeJSON(w, http.StatusAccepted, map[string]any{
			"error":    "postfix unavailable, saved as draft",
			"draft_id": errSavedAsDraft.DraftID,
		})
		return
	}

	var errRecipientNotFound *service.ErrRecipientNotFound
	if errors.As(err, &errRecipientNotFound) {
		response.NotFoundWithMessage(w, "recipient not found: "+errRecipientNotFound.Email)
		return
	}

	switch {
	case errors.Is(err, service.ErrConflict),
		errors.Is(err, service.ErrDraftsLimit):
		response.StatusConflict(w)
	case errors.Is(err, service.ErrBadRequest),
		errors.Is(err, service.ErrEmptyIDs),
		errors.Is(err, service.ErrDraftValidation),
		errors.Is(err, service.ErrDraftNotReady),
		errors.Is(err, service.ErrStorageUnavailable):
		response.BadRequest(w)
	case errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrEmailNotFound),
		errors.Is(err, service.ErrNoValidReceivers),
		errors.Is(err, service.ErrAttachmentNotFound):
		response.NotFound(w)
	case errors.Is(err, service.ErrAccessDenied):
		response.Forbidden(w)
	default:
		response.InternalError(w)
	}
}
