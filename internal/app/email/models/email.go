package models

import "time"

type Email struct {
	ID        int64     `json:"email_id"`
	SenderID  int64     `json:"from"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type UserEmail struct {
	ID         int64
	EmailID    int64
	ReceiverID int64
	IsRead     bool
}
