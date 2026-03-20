package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type Service interface {
	GetEmailsByReceiver(ctx context.Context, email string) ([]models.Email, error)
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Success      200  {array}   models.Email
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       api/v1/emails [get]
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

	result, err := handler.service.GetEmailsByReceiver(r.Context(), payload.Email)
	if err != nil {
		response.InternalError(w)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		response.InternalError(w)
		return
	}
}
