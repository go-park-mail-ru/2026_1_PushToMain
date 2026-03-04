package handler

import (
    "encoding/json"
    "net/http"
)

type ErrorResponse struct {
    Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func BadRequest(w http.ResponseWriter, msg string) {
    WriteJSON(w, http.StatusBadRequest, ErrorResponse{Message: msg})
}