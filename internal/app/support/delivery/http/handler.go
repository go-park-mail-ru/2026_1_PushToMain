package http

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/supoort/service"
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

	// Private routes
	private.HandleFunc("/support/send", h.SendQuestion).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/support/myqyestions", h.GetMyQuestions).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/support/changestatus", h.ChangeStatus).Methods(http.MethodPut, http.MethodOptions)
	private.HandleFunc("/support/answer", h.AnswerOnQuestion).Methods(http.MethodPost, http.MethodOptions)
	private.HandleFunc("/support/{id}/chat", h.GetAllMessages).Methods(http.MethodGet, http.MethodOptions)
	private.HandleFunc("/support/admin/questions", h.GetAllQuestionsByFilter).Methods(http.MethodGet, http.MethodOptions)

}

func parseCommonErrors(err error, w http.ResponseWriter) {
	switch {

	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(w)

	case errors.Is(err, service.ErrWrongPassword):
		response.Unauthorized(w)

	case errors.Is(err, service.ErrUserAlreadyExists):
		response.StatusConflict(w)

	default:
		response.InternalError(w)
	}
}
