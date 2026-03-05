package main

import (
	"auth/internal/handlers"
	"auth/internal/middleware"
	"auth/internal/repository"
	"auth/internal/service"
	"net/http"
)

func main() {
	repo := repository.NewMemoryUserRepo()
	svc := service.NewAuthService(repo)
	handler := handlers.NewAuthHandler(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("/signup", handler.SignUp)
	mux.HandleFunc("/signin", handler.SignIn)

	http.ListenAndServe(":8080", middleware.CORS(mux))
}
