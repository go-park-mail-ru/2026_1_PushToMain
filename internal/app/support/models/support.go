package models

import "time"

type Ticket struct {
	ID        int64
	UserID    int64
	SupportID *int64
	Subject   string
	Category  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
}

type Message struct {
	ID        int64
	TicketID  int64
	AuthorID  int64
	Body      string
	CreatedAt time.Time
}

type TicketStats struct {
	Total      int
	ByStatus   map[string]int
	ByCategory map[string]int
}
