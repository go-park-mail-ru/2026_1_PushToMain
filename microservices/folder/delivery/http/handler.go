//go:generate mockgen -destination=../../../../mocks/app/folder/mock_handler_service.go -package=mocks github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/delivery/http Service
package handler

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/service"
	"github.com/gorilla/mux"
)

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	private.HandleFunc("/folder/new", h.CreateNewFolder).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/folder/{folderID}/name", h.ChangeFolderName).Methods(http.MethodPut, http.MethodOptions)

	private.HandleFunc("/folder/{folderID}", h.GetEmailsFromFolder).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/folder/{folderID}/add", h.AddEmailsInFolder).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/folder/{folderID}/delete", h.DeleteEmailsFromFolder).Methods(http.MethodDelete, http.MethodOptions)
	private.HandleFunc("/folder/{folderID}", h.DeleteFolder).Methods(http.MethodDelete, http.MethodOptions)
}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, service.ErrEmptyEmailsList):
		response.BadRequest(w)

	case errors.Is(err, service.ErrFolderNotFound):
		response.NotFound(w)

	case errors.Is(err, service.ErrFolderAlreadyExists):
		response.StatusConflict(w)

	case errors.Is(err, service.ErrMaxFoldersReached):
		response.StatusConflict(w)

	case errors.Is(err, service.ErrAccessDenied):
		response.Forbidden(w)

	default:
		response.InternalError(w)
	}
}
