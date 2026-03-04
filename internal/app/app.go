package app

import (
	"net/http"
	"smail/internal/app/handler"
)

type App struct {

}

func New() *App {
	return &App{

	}
}

func (app *App) Run() {
	handler := handler.NewHandler()
	router := handler.InitRoutes()

	http.ListenAndServe(":8087", router)
}

