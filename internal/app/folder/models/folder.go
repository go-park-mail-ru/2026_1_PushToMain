package models

import "time"

type Folder struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EmailFromFolder struct {
	ID            int64
	SenderEmail   string
	SenderName    string
	SenderSurname string
	ReceiverList  []string
	Header        string
	Body          string
	CreatedAt     time.Time
	IsRead        bool
	IsFavorite    bool
}
