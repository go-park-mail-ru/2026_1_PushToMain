package main

import (
	"auth/internal/handlers"
	"auth/internal/repository"
	"auth/internal/service"
	"net/http"
)

func main() {
	repo := repository.NewMemoryUserRepo()
	svc := service.NewAuthService(repo)
	handler := handlers.NewAuthHandler(svc)

	http.HandleFunc("/signup", handler.SignUp)
	http.HandleFunc("/signin", handler.SignIn)

	http.ListenAndServe(":8080", nil)
}
