package handler

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

var EmailsMock []models.Email = []models.Email{
	{
		EmailID: "email-001",
		From:    "ivan.petrov@gmail.com",
		To:      []string{"anna.sidorova@smail.ru", "dmitry.kozlov@smail.ru"},
		Header:  "Встреча в пятницу",
		Body:    "Привет! Напоминаю про встречу в пятницу в 15:00. Не забудьте взять ноутбуки.",
	},
	{
		EmailID: "email-002",
		From:    "anna.sidorova@mail.ru",
		To:      []string{"ivan.petrov@smail.ru"},
		Header:  "Отчёт за март",
		Body:    "Высылаю отчёт за март. Все показатели в норме, подробности внутри.",
	},
	{
		EmailID: "email-003",
		From:    "sergey.volkov@company.ru",
		To:      []string{"ivan.petrov@smail.ru", "anna.sidorova@smail.ru", "dmitry.kozlov@smail.ru"},
		Header:  "Новый дизайн главной страницы",
		Body:    "Команда, посмотрите новый макет. Жду фидбек до конца дня.",
	},
	{
		EmailID: "email-004",
		From:    "dmitry.kozlov@yandex.ru",
		To:      []string{"sergey.volkov@smail.ru"},
		Header:  "Re: Новый дизайн главной страницы",
		Body:    "Выглядит хорошо, но кнопка CTA слишком маленькая на мобиле.",
	},
	{
		EmailID: "email-005",
		From:    "olga.novikova@inbox.ru",
		To:      []string{"ivan.petrov@smail.ru"},
		Header:  "Баг в продакшне",
		Body:    "Срочно! Упал сервис авторизации, пользователи не могут войти. Смотрю логи.",
	},
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых авторизованный пользователь указан получателем
// @Tags         emails
// @Produce      json
// @Success      200  {array}   models.Email
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     BearerAuth
// @Router       /emails [get]
func (h *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	if payload.Email == "" {
		response.BadRequest(w)
		return
	}

	result := make([]models.Email, 0)
	for _, email := range EmailsMock {
		for _, to := range email.To {
			if to == payload.Email {
				result = append(result, email)
				break
			}
		}
	}

	// TODO: Replace WriteJSON
	response.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) GetFullEmailByID(w http.ResponseWriter, r *http.Request) {

}
