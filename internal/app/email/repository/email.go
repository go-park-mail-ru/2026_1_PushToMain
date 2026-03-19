package repository

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

type MemoryRepo struct {
	emails []models.Email
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		emails: []models.Email{
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
		},
	}
}

type Repository interface {
	GetAll(ctx context.Context) ([]models.Email, error)
}

/*func NewEmailRepo(data []models.Email) *MemoryEmailRepo {
	return &MemoryEmailRepo{emails: data}
}*/

func (r *MemoryRepo) GetAll(ctx context.Context) ([]models.Email, error) {
	return r.emails, nil
}
