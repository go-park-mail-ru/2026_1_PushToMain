package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
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
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	private.HandleFunc("/emails", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/myemails", h.GetMyEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/send", h.SendEmail).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/forward", h.ForwardEmail).Methods(http.MethodPost, http.MethodOptions)

	private.HandleFunc("/emails/spam", h.GetSpamEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/trash", h.GetTrashEmails).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/read", h.MarkEmailsAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/unread", h.MarkEmailsAsUnRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id}", h.GetEmailByID).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id}/read", h.MarkEmailAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id}/unread", h.MarkEmailAsUnRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id}/folder", h.ChangeFolder).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id}/restore", h.RestoreFromTrash).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/delete", h.DeleteEmailForReceiver).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/myemails/delete", h.DeleteEmailForSender).Methods(http.MethodDelete, http.MethodOptions)

	// Drafts
	private.HandleFunc("/drafts", h.CreateDraft).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/drafts", h.GetDrafts).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/drafts/{id}", h.GetDraftByID).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/drafts/{id}", h.UpdateDraft).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/drafts/{id}", h.DeleteDraft).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/drafts/{id}/send", h.SendDraft).Methods(http.MethodPost, http.MethodOptions)
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, service.ErrConflict):
		response.StatusConflict(w)
	case errors.Is(err, service.ErrBadRequest):
		response.BadRequest(w)
	case errors.Is(err, service.ErrInvalidFolder):
		response.BadRequest(w)
	case errors.Is(err, service.ErrDraftValidation):
		response.BadRequest(w)
	case errors.Is(err, service.ErrDraftNotReady):
		response.BadRequest(w)
	case errors.Is(err, service.ErrDraftsLimit):
		response.StatusConflict(w)
	case errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrEmailNotFound),
		errors.Is(err, service.ErrFolderNotFound),
		errors.Is(err, service.ErrNoValidReceivers):
		response.NotFound(w)
	case errors.Is(err, service.ErrAccessDenied):
		response.Forbidden(w)
	default:
		response.InternalError(w)
	}
}
