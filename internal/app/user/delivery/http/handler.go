package http

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Config struct {
	TTL time.Duration
}

type Handler struct {
	service Service
	cfg     Config
}

func New(service Service, cfg Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) InitRoutes(public, private *mux.Router) {
	public.HandleFunc("/signup", h.SignUp).Methods(http.MethodPost, http.MethodOptions)
	public.HandleFunc("/signin", h.SignIn).Methods(http.MethodPost, http.MethodOptions)
	public.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)
	public.HandleFunc("/logout", h.Logout).Methods(http.MethodPost, http.MethodOptions)
	public.HandleFunc("/csrf", h.GetCSRF).Methods(http.MethodGet, http.MethodOptions)
}
