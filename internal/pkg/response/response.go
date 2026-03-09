package response

import (
    "encoding/json"
    "net/http"
    "fmt"
)

type ErrorResponse struct {
    Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func BadRequest(w http.ResponseWriter) {
    WriteJSON(w, http.StatusBadRequest, ErrorResponse{Message: "Bad request"})
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
