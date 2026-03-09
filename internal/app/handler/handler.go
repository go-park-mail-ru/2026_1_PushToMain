package handler

import (
	"github.com/gorilla/mux"
	"net/http"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Handler struct {
	authService AuthService
}

func NewHandler(service AuthService) *Handler {
	return &Handler{authService: service}
}

func (h *Handler) InitRoutes(router *mux.Router) {
	router.HandleFunc("/signup", h.SignUp).Methods(http.MethodPost, http.MethodOptions)
	router.HandleFunc("/signin", h.SignIn).Methods(http.MethodPost, http.MethodOptions)
	router.HandleFunc("/inbox", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
	router.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)
}