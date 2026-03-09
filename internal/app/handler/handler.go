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

func (h *Handler) InitRoutes(public, private *mux.Router) {
	public.HandleFunc("/signup", h.SignUp).Methods(http.MethodPost, http.MethodOptions)
	public.HandleFunc("/signin", h.SignIn).Methods(http.MethodPost, http.MethodOptions)
	public.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	private.HandleFunc("/emails", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
}