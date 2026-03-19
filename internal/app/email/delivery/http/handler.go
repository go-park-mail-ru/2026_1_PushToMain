package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Handler struct {
	service Service

	ttl time.Duration
}

func NewHandler(service Service, ttl time.Duration) *Handler {
	return &Handler{service: service, ttl: ttl}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	private.HandleFunc("/emails", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
}
