package response

import (
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func BadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{ "error": "Bad request" }`)
}

func Unauthorized(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprintf(w, `{ "error": "Unauthorized" }`)
}

func InternalError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{ "error": "Internal server error" }`)
}

func StatusConflict(w http.ResponseWriter) {
	w.WriteHeader(http.StatusConflict)
	fmt.Fprintf(w, `{ "error": "User already exsist" }`)
}

func Forbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, `{ "error": "Don't have access" }`)
}

func NotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{ "error": "Not found" }`)
}
