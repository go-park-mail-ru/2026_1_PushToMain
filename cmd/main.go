package main

import (
	"net/http"
	"smail/internal/app/handler"
)

func main() {
	h := handler.NewHandler()
	r := h.InitRoutes()

	http.ListenAndServe(":8090", r)
}