package models

import "time"

type Email struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	CreatedAt time.Time
}

type EmailWithMetadata struct {
	Email
	IsRead          bool
	ReceivedAt      time.Time
	ReceiversEmails []string
}

type EmailWithAvatar struct {
	Email
	SenderImagePath string
}

type UserEmail struct {
	ID         int64
	EmailID    int64
	ReceiverID int64
	IsRead     bool
}

type User struct {
	ID       int64
	Email    string
	Password string
	Name     string
	Surname  string
}
