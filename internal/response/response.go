package response

import (
	"fmt"
	"net/http"
)

func BadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{ "error": "BAD_REQUEST" }`)
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
