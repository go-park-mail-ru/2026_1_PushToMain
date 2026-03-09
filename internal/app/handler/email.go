package handler

import (
	"net/http"
	"encoding/json"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/models"
    "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/response"
)

var EmailsMock []models.Email = []models.Email{
    {
        EmailID: "email-001",
        From:    "ivan.petrov@gmail.com",
        To:      []models.EmailName{"anna.sidorova@mail.ru", "dmitry.kozlov@yandex.ru"},
        Header:  "Встреча в пятницу",
        Body:    "Привет! Напоминаю про встречу в пятницу в 15:00. Не забудьте взять ноутбуки.",
    },
    {
        EmailID: "email-002",
        From:    "anna.sidorova@mail.ru",
        To:      []models.EmailName{"ivan.petrov@gmail.com"},
        Header:  "Отчёт за март",
        Body:    "Высылаю отчёт за март. Все показатели в норме, подробности внутри.",
    },
    {
        EmailID: "email-003",
        From:    "sergey.volkov@company.ru",
        To:      []models.EmailName{"ivan.petrov@gmail.com", "anna.sidorova@mail.ru", "dmitry.kozlov@yandex.ru"},
        Header:  "Новый дизайн главной страницы",
        Body:    "Команда, посмотрите новый макет. Жду фидбек до конца дня.",
    },
    {
        EmailID: "email-004",
        From:    "dmitry.kozlov@yandex.ru",
        To:      []models.EmailName{"sergey.volkov@company.ru"},
        Header:  "Re: Новый дизайн главной страницы",
        Body:    "Выглядит хорошо, но кнопка CTA слишком маленькая на мобиле.",
    },
    {
        EmailID: "email-005",
        From:    "olga.novikova@inbox.ru",
        To:      []models.EmailName{"ivan.petrov@gmail.com"},
        Header:  "Баг в продакшне",
        Body:    "Срочно! Упал сервис авторизации, пользователи не могут войти. Смотрю логи.",
    },
}

type EmailRequest struct {
	Owner models.EmailName `json:"owner"`
}

// @Summary      Получить письма пользователя
// @Description  Возвращает список писем, в которых пользователь указан получателем
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        input body handler.EmailRequest true "Владелец почты"
// @Success      200    {array}   models.Email
// @Router       /emails [get]
func (h *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {
    var emailRequest EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&emailRequest); err != nil {
        response.BadRequest(w)
        return
    }

    if emailRequest.Owner == "" {
        response.BadRequest(w)
        return
    }

	result := make([]models.Email, 0)
	for _, email := range EmailsMock {
		for _, to := range email.To {
			if to == emailRequest.Owner {
				result = append(result, email)
				break
			}
		}
	}

	response.WriteJSON(w, http.StatusOK, result)
}	

func (h *Handler) GetFullEmailByID(w http.ResponseWriter, r *http.Request) {

}