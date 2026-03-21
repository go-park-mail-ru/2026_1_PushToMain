package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
)

type Repository struct {
	emails     []models.Email
	user_email []models.User_email
}

func New() *Repository {
	return &Repository{
		emails: []models.Email{
			{
				ID:        1,
				SenderID:  1,
				Header:    "Встреча в пятницу",
				Body:      "Привет! Напоминаю про встречу в пятницу в 15:00. Не забудьте взять ноутбуки.",
				CreatedAt: time.Date(2025, time.March, 22, 15, 30, 0, 0, time.UTC),
			},
			{
				ID:       2,
				SenderID: 2,
				Header:   "Отчёт за март",
				Body:     "Высылаю отчёт за март. Все показатели в норме, подробности внутри.",
			},
			{
				ID:       3,
				SenderID: 2,
				Header:   "Новый дизайн главной страницы",
				Body:     "Команда, посмотрите новый макет. Жду фидбек до конца дня.",
			},
			{
				ID:       4,
				SenderID: 3,
				Header:   "Re: Новый дизайн главной страницы",
				Body:     "Выглядит хорошо, но кнопка CTA слишком маленькая на мобиле.",
			},
			{
				ID:       5,
				SenderID: 4,
				Header:   "Баг в продакшне",
				Body:     "Срочно! Упал сервис авторизации, пользователи не могут войти. Смотрю логи.",
			},
		},

		user_email: []models.User_email{
			{
				ID:         1,
				EmailID:    1,
				ReceiverID: 1,
				IsRead:     false,
			},
			{
				ID:         2,
				EmailID:    2,
				ReceiverID: 1,
				IsRead:     false,
			},
			{
				ID:         3,
				EmailID:    2,
				ReceiverID: 1,
				IsRead:     false,
			},
			{
				ID:         4,
				EmailID:    3,
				ReceiverID: 1,
				IsRead:     false,
			},
			{
				ID:         5,
				EmailID:    4,
				ReceiverID: 1,
				IsRead:     false,
			},
		},
	}
}

func (r *Repository) GetEmailsByReceiver(ctx context.Context, userID int64) ([]models.Email, error) {

	result := make([]models.Email, 0)

	userEmailIDs := make(map[int64]bool)
	for _, userEmail := range r.user_email {
		if userEmail.ReceiverID == userID {
			userEmailIDs[userEmail.EmailID] = true
		}
	}

	for _, email := range r.emails {
		if userEmailIDs[email.ID] {
			result = append(result, email)
		}
	}

	return result, nil
}
