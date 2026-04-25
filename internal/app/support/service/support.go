package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/support/models"
)

type Repository interface {
	CreateTicket(ctx context.Context, ticket models.Ticket) (int64, error)
	GetTicketByTicketID(ctx context.Context, ticketID int64) (*models.Ticket, error)
	GetAllTicketsByUserID(ctx context.Context, userID int64) ([]models.Ticket, error)
	UpdateTicketStatus(ctx context.Context, ticketID int64, newStatus string) error
	ListAllTickets(ctx context.Context, status, theme string) ([]models.Ticket, error)
	CreateMessage(ctx context.Context, msg models.Message) (int64, error)
	ListMessagesByTicket(ctx context.Context, ticketID int64) ([]models.Message, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}
type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{repo: repo}
}

type SendQuestionInput struct {
	UserID int64
	Theme  string
	Header string
	Body   string
}

type Question struct {
	Theme    string
	Header   string
	TickerID int64
	Status   string
}

type GetMyQuestionsInput struct {
	UserID int64
}

type GetMyQuestionsResult struct {
	Questions []Question
}

type ChangeStatusInput struct {
	UserID     int64
	Status     string
	QuestionID int64
}
type AnswerOnQuestionInput struct {
	UserID     int64
	QuestionID int64
	Answer     string
}

type GettAllMessagesInput struct {
	UserID     int64
	QuestionID int64
}
type Message struct {
	IsAdmin bool
	Text    string
}
type GettAllMessagesResult struct {
	Messages []Message
}

type GetAllQuestionsByFilterInput struct {
	Theme  string
	Status string
	UserID int64
}

type GetAllQuestionsByFilterResult struct {
	Questions []Question
}

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrAccessDenied   = errors.New("access denied")
	ErrInvalidStatus  = errors.New("invalid status")
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrNotAdmin       = errors.New("user is not admin")
)

// SendQuestion создаёт новый тикет и первое сообщение
func (s *Service) SendQuestion(ctx context.Context, input SendQuestionInput) (string, error) {
	if input.Body == "" {
		return "", ErrEmptyMessage
	}

	// Создаём тикет
	ticket := models.Ticket{
		UserID:   input.UserID,
		Subject:  input.Header,
		Category: input.Theme,
	}

	ticketID, err := s.repo.CreateTicket(ctx, ticket)
	if err != nil {
		return "", fmt.Errorf("failed to create ticket: %w", err)
	}

	// Создаём первое сообщение
	msg := models.Message{
		TicketID: ticketID,
		AuthorID: input.UserID,
		Body:     input.Body,
	}

	if _, err := s.repo.CreateMessage(ctx, msg); err != nil {
		return "", fmt.Errorf("failed to create message: %w", err)
	}

	return fmt.Sprintf("Ticket #%d created", ticketID), nil
}

// GetMyQuestions возвращает все тикеты пользователя
func (s *Service) GetMyQuestions(ctx context.Context, input GetMyQuestionsInput) (*GetMyQuestionsResult, error) {
	tickets, err := s.repo.GetAllTicketsByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tickets: %w", err)
	}

	result := make([]Question, len(tickets))
	for i, ticket := range tickets {
		result[i] = Question{
			Theme:    ticket.Category,
			Header:   ticket.Subject,
			TickerID: ticket.ID,
			Status:   ticket.Status,
		}
	}

	return &GetMyQuestionsResult{Questions: result}, nil
}

// ChangeStatus изменяет статус тикета (только для админов)
func (s *Service) ChangeStatus(ctx context.Context, input ChangeStatusInput) error {
	// Проверяем, что пользователь — админ
	isAdmin, err := s.repo.IsAdmin(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to check admin: %w", err)
	}
	if !isAdmin {
		return ErrNotAdmin
	}

	// Валидация статуса
	validStatuses := map[string]bool{"open": true, "in_progress": true, "closed": true}
	if !validStatuses[input.Status] {
		return ErrInvalidStatus
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetTicketByTicketID(ctx, input.QuestionID)
	if err != nil {
		return ErrTicketNotFound
	}

	if ticket.Status == input.Status {
		return nil // статус не изменился
	}

	return s.repo.UpdateTicketStatus(ctx, input.QuestionID, input.Status)
}

// AnswerOnQuestion добавляет ответ на вопрос (только для админов)
func (s *Service) AnswerOnQuestion(ctx context.Context, input AnswerOnQuestionInput) error {
	if input.Answer == "" {
		return ErrEmptyMessage
	}

	// Проверяем, что пользователь — админ
	isAdmin, err := s.repo.IsAdmin(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to check admin: %w", err)
	}
	if !isAdmin {
		return ErrNotAdmin
	}

	// Проверяем существование тикета
	ticket, err := s.repo.GetTicketByTicketID(ctx, input.QuestionID)
	if err != nil {
		return ErrTicketNotFound
	}

	// Создаём сообщение от админа
	msg := models.Message{
		TicketID: input.QuestionID,
		AuthorID: input.UserID, // это админ
		Body:     input.Answer,
	}

	if _, err := s.repo.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	// Если статус был "open", меняем на "in_progress"
	if ticket.Status == "open" {
		_ = s.repo.UpdateTicketStatus(ctx, input.QuestionID, "in_progress")
	}

	return nil
}

// GetAllMessages возвращает все сообщения по тикету
func (s *Service) GetAllMessages(ctx context.Context, input GettAllMessagesInput) (*GettAllMessagesResult, error) {
	// Проверяем существование тикета
	ticket, err := s.repo.GetTicketByTicketID(ctx, input.QuestionID)
	if err != nil {
		return nil, ErrTicketNotFound
	}

	// Проверяем права доступа: либо автор тикета, либо админ
	isAdmin, _ := s.repo.IsAdmin(ctx, input.UserID)
	if ticket.UserID != input.UserID && !isAdmin {
		return nil, ErrAccessDenied
	}

	// Получаем сообщения
	messages, err := s.repo.ListMessagesByTicket(ctx, input.QuestionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	result := make([]Message, len(messages))
	for i, msg := range messages {
		// Определяем, от админа ли сообщение
		isMsgFromAdmin, _ := s.repo.IsAdmin(ctx, msg.AuthorID)
		result[i] = Message{
			IsAdmin: isMsgFromAdmin,
			Text:    msg.Body,
		}
	}

	return &GettAllMessagesResult{Messages: result}, nil
}

// GetAllQuestionsByFilter возвращает все тикеты с фильтрацией (только для админов)
func (s *Service) GetAllQuestionsByFilter(ctx context.Context, input GetAllQuestionsByFilterInput) (*GetAllQuestionsByFilterResult, error) {
	// Проверяем, что пользователь — админ
	isAdmin, err := s.repo.IsAdmin(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin: %w", err)
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	tickets, err := s.repo.ListAllTickets(ctx, input.Status, input.Theme)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	result := make([]Question, len(tickets))
	for i, ticket := range tickets {
		result[i] = Question{
			Theme:    ticket.Category,
			Header:   ticket.Subject,
			TickerID: ticket.ID,
			Status:   ticket.Status,
		}
	}

	return &GetAllQuestionsByFilterResult{Questions: result}, nil
}
