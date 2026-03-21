package models

import "time"

type Email struct {
	ID        int64     `json:"email-id"`
	SenderID  int64     `json:"from"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created-at"`
}
