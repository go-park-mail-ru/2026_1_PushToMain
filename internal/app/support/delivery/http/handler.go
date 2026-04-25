package handler

import (
	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/support/service"
)

type Handler struct {
	service *service.Service
}

func New(service *service.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) InitRoutes(private, admin *mux.Router) {
}
