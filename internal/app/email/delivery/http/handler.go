package handler

import (
	"net/http"
	"time"

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
	private.HandleFunc("/send", h.SendEmail).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/forward", h.ForwardEmail).Methods(http.MethodPost, http.MethodOptions)

	private.HandleFunc("/emails/{id:[0-9]+}", h.GetEmailByID).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/emails/{id:[0-9]+}/read", h.MarkEmailAsRead).Methods(http.MethodPut, http.MethodOptions)
}
