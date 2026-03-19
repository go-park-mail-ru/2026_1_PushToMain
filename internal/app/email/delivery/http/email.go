package handler

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type EmailService interface {
	GetEmailsByReceiver(ctx context.Context, email string) ([]models.Email, error)
}

func NewEmailHandler(s EmailService) *Handler {
	return &Handler{EmailService: s}
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Success      200  {array}   models.Email
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     BearerAuth
// @Router       /emails [get]
func (handler *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	if payload.Email == "" {
		response.BadRequest(w)
		return
	}

	result, err := handler.EmailService.GetEmailsByReceiver(r.Context(), payload.Email)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.WriteJSON(w, http.StatusOK, result)
}
