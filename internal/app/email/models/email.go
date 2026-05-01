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
	IsStarred       bool
	IsSpam          bool
	IsDeleted       bool
	ReceivedAt      time.Time
	ReceiversEmails []string
}

type EmailWithAvatar struct {
	Email
	SenderImagePath string
	ReceiversEmails []string
}

type Draft struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	Receivers []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserEmail struct {
	ID        int64
	EmailID   int64
	UserID    int64
	IsSender  bool
	IsRead    bool
	IsDeleted bool
	IsStarred bool
	IsSpam    bool
	IsDraft   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID       int64
	Email    string
	Password string
	Name     string
	Surname  string
}
