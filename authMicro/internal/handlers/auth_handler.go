package handlers

import (
	"auth/internal/service"
	"encoding/json"
	"net/http"
)

type SignUpRequest struct {
	Name           string `json:"name"`
	Surname        string `json:"surname"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"passwordRepeat"`
}
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type AuthResponse struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (handler *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cmd := service.SignUpCommand{
		Email:          req.Email,
		Password:       req.Password,
		PasswordRepeat: req.PasswordRepeat,
		Name:           req.Name,
		Surname:        req.Surname,
	}

	token, err := handler.service.SignUp(r.Context(), cmd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

func (handler *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cmd := service.SignInCommand{
		Email:    req.Email,
		Password: req.Password,
	}

	token, err := handler.service.SignIn(r.Context(), cmd)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}
