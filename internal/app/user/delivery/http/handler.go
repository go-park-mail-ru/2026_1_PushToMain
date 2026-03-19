package http

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Handler struct {
	service Service

	ttl time.Duration
}

func NewHandler(service Service, ttl time.Duration) *Handler {
	return &Handler{service: service, ttl: ttl}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	public.HandleFunc("/signup", h.SignUp).Methods(http.MethodPost, http.MethodOptions)
	public.HandleFunc("/signin", h.SignIn).Methods(http.MethodPost, http.MethodOptions)
	public.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)
	public.HandleFunc("/logout", h.Logout).Methods(http.MethodPost, http.MethodOptions)
}
