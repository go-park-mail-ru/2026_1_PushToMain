package handler

import (
	"github.com/gorilla/mux"
	"net/http"
	httpSwagger "github.com/swaggo/http-swagger"
	_ "smail/docs"
)

type Handler struct {

}

func NewHandler() *Handler {
	return &Handler{

	}
}


func (h *Handler) InitRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/inbox/", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
<<<<<<< Updated upstream
=======
	r.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
>>>>>>> Stashed changes
	
	return r
}