package handler

import (
	"github.com/gorilla/mux"
	"net/http"
)

type Handler struct {

}

func NewHandler() *Handler {
	return &Handler{

	}
}


func (h *Handler) InitRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/inbox/", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
	
	return r
}