package response

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) error {
	w.WriteHeader(status)
	//что с этой ошибкой по итогу делать?
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return err
	}
	return nil
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
