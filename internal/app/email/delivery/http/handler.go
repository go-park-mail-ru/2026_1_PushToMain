package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	EmailService EmailService

	ttl time.Duration
}

func NewHandler(service EmailService, ttl time.Duration) *Handler {
	return &Handler{EmailService: service, ttl: ttl}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	private.HandleFunc("/emails", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
}
