package response

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

func BadRequest(w http.ResponseWriter) {
	WriteJSON(w, http.StatusBadRequest, ErrorResponse{
		Message: "Bad request",
	})
}

func Unauthorized(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized, ErrorResponse{
		Message: "Unauthorized",
	})
}

func InternalError(w http.ResponseWriter) {
	WriteJSON(w, http.StatusInternalServerError, ErrorResponse{
		Message: "Internal server error",
	})
}

func StatusConflict(w http.ResponseWriter) {
	WriteJSON(w, http.StatusConflict, ErrorResponse{
		Message: "User already exsist",
	})
}

// чтоб в handler не писать write json и статус ok, а только полезуню data и responsewriter закидывать
func OK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}
