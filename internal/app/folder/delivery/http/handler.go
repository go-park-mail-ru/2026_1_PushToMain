package handler

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
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

}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {

	default:
		response.InternalError(w)
	}
}
