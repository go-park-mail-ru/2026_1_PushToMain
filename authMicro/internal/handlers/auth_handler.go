package handlers

import (
	"auth/internal/dto"
	"auth/internal/service"
	"encoding/json"
	"net/http"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (handler *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req dto.SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := handler.service.SignUp(
		req.Email,
		req.Password,
		req.PasswordRepeat,
		req.Name,
		req.Surname,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	json.NewEncoder(w).Encode(dto.AuthResponse{Token: token})
}

func (handler *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req dto.SignInRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	token, err := handler.service.SignIn(req.Email, req.Password)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(dto.AuthResponse{Token: token})
}
