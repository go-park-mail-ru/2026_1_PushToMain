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

	private.HandleFunc("/emails/{id}", h.GetEmailByID).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id}/read", h.MarkEmailAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/{id}/unread", h.MarkEmailAsUnRead).Methods(http.MethodPut, http.MethodOptions)

	private.HandleFunc("/emails/read", h.MarkEmailsAsRead).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/emails/unread", h.MarkEmailsAsUnRead).Methods(http.MethodPut, http.MethodOptions)

	private.HandleFunc("/emails/delete", h.DeleteEmailForReceiver).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/myemails/delete", h.DeleteEmailForSender).Methods(http.MethodDelete, http.MethodOptions)
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, service.ErrConflict):
		response.StatusConflict(w)

	case errors.Is(err, service.ErrBadRequest):
		response.BadRequest(w)

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
