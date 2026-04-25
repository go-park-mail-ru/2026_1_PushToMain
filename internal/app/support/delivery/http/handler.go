package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/support/service"
)

type Config struct {
	TTL time.Duration
}

type Handler struct {
	service Service
}

func New(service Service, cfg Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {

}

func parseCommonErrors(err error, w http.ResponseWriter) {
}
