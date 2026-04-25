package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/support/models"
)

const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

var (
	ErrQueryError     = errors.New("failed to exec query")
	ErrTicketNotFound = errors.New("ticket not found")
)

type Repository struct {
	db *sql.DB
}

type ListTicketsFilter struct {
	Status   *string
	Category *string
}

// GetTicketByIDForUser(ctx, ticketID, userID) (*Ticket, error) // для юзера
// ListTicketsByUser(ctx, userID, limit, offset) ([]Ticket, error)
// CountTicketsByUser(ctx, userID) (int, error)
// ListAllTickets(ctx, filter, limit, offset) ([]Ticket, error)
// UpdateTicketStatus(ctx, ticketID, newStatus) error

// // Сообщения
// CreateMessage(ctx, tx, msg) (int64, error)
// ListMessagesByTicket(ctx, ticketID, limit, offset) ([]Message, error)

// // Админ
// IsAdmin(ctx, userID) (bool, error)
// GetStats(ctx) (*TicketStats, error)

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *Repository) CreateTicket(ctx context.Context, ticket models.Ticket) (int64, error) {
	query := `
		INSERT
		INTO support_tickets
		(user_id, subject, category)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var ticketID int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		ticket.UserID,
		ticket.Subject,
		ticket.Category,
	).Scan(&ticketID)

	if err != nil {
		return 0, ErrQueryError
	}

	return ticketID, nil
}

func (r *Repository) GetTicketByTicketID(ctx context.Context, ticketID int64) (*models.Ticket, error) {
	query := `
		SELECT id, user_id, support_id, subject, category, status, created_at, updated_at, closed_at
		FROM support_tickets
		WHERE id = $1
	`

	var ticket models.Ticket
	err := r.db.QueryRowContext(ctx, query, ticketID).
		Scan(
			&ticket.ID,
			&ticket.UserID,
			&ticket.SupportID,
			&ticket.Subject,
			&ticket.Category,
			&ticket.Status,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
			&ticket.ClosedAt,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTicketNotFound
	}
	if err != nil {
		return nil, ErrQueryError
	}

	return &ticket, nil
}

func (r *Repository) GetAllTicketsByUserID(ctx context.Context, userID int64, limit int, offset int) ([]models.Ticket, error) {
	query := `
		SELECT id, user_id, support_id, subject, category, status, created_at, updated_at, closed_at
		FROM support_tickets
		WHERE user_id = $1
		ORDER_BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, ErrQueryError
	}

	defer rows.Close()
	return scanTickets(rows)
}

func (r *Repository) UpdateTicketStatus(ctx context.Context, ticketID int64, newStatus string) error {
	query := `
        UPDATE support_tickets
        SET status = $1, closed_at = CASE WHEN $1 = 'closed' THEN NOW() ELSE closed_at END
        WHERE id = $2
    `

	result, err := r.db.ExecContext(ctx, query, newStatus, ticketID)
	if err != nil {
		return ErrQueryError
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ErrQueryError
	}
	if rowsAffected == 0 {
		return ErrTicketNotFound
	}

	return nil
}

func (r *Repository) ListAllTickets(ctx context.Context, filter ListTicketsFilter, limit, offset int) ([]models.Ticket, error) {
	var (
		conds []string
		args  []interface{}
	)

	if filter.Status != nil {
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *filter.Status)
	}
	if filter.Category != nil {
		conds = append(conds, fmt.Sprintf("category = $%d", len(args)+1))
		args = append(args, *filter.Category)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	query := fmt.Sprintf(`
        SELECT id, user_id, support_id, subject, category, status, created_at, updated_at, closed_at
        FROM support_tickets
        %s
        ORDER BY created_at DESC
        LIMIT $%d OFFSET $%d
    `, where, len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ErrQueryError
	}
	defer rows.Close()

	return scanTickets(rows)
}

func (r *Repository) CreateMessage(ctx context.Context, msg models.Message) (int64, error) {
	query := `
        INSERT INTO support_messages (ticket_id, author_id, body)
        VALUES ($1, $2, $3)
        RETURNING id
    `

	var messageID int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		msg.TicketID,
		msg.AuthorID,
		msg.Body,
	).Scan(&messageID)

	if err != nil {
		return 0, ErrQueryError
	}

	return messageID, nil
}

func (r *Repository) ListMessagesByTicket(ctx context.Context, ticketID int64, limit, offset int) ([]models.Message, error) {
	query := `
        SELECT id, ticket_id, author_id, body, created_at
        FROM support_messages
        WHERE ticket_id = $1
        ORDER BY id
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.QueryContext(ctx, query, ticketID, limit, offset)
	if err != nil {
		return nil, ErrQueryError
	}
	defer rows.Close()

	messages := make([]models.Message, 0)
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(
			&m.ID,
			&m.TicketID,
			&m.AuthorID,
			&m.Body,
			&m.CreatedAt,
		); err != nil {
			return nil, ErrQueryError
		}
		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, ErrQueryError
	}

	return messages, nil
}

func (r *Repository) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM support_admins WHERE user_id = $1)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, ErrQueryError
	}

	return exists, nil
}

func scanTickets(rows *sql.Rows) ([]models.Ticket, error) {
	tickets := make([]models.Ticket, 0)
	for rows.Next() {
		var t models.Ticket
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.SupportID,
			&t.Subject,
			&t.Category,
			&t.Status,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.ClosedAt,
		); err != nil {
			return nil, ErrQueryError
		}
		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, ErrQueryError
	}

	return tickets, nil
}
