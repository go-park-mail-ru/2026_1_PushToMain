package handler

import (
	"net/http"
	"encoding/json"
	"smail/internal/app/models"
    "smail/internal/app/response"
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

var EmailRequest struct {
	Owner models.EmailName `json:"owner"`
}

func (h *Handler) GetEmails(w http.ResponseWriter, r *http.Request) {
	if err := json.NewDecoder(r.Body).Decode(&EmailRequest); err != nil {
        response.BadRequest(w)
        return
    }

    if EmailRequest.Owner == "" {
        response.BadRequest(w)
        return
    }

	result := make([]models.Email, 0)
	for _, email := range EmailsMock {
		for _, to := range email.To {
			if to == EmailRequest.Owner {
				result = append(result, email)
				break
			}
		}
	}

	response.WriteJSON(w, http.StatusOK, result)
}	

func (h *Handler) GetFullEmailByID(w http.ResponseWriter, r *http.Request) {

}