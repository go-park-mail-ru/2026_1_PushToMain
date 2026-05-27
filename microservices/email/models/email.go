package models

import "time"

type MailboxStats struct {
	Total  int
	Unread int
}

type Email struct {
	ID          int64
	SenderID    *int64
	SenderEmail string
	Header      string
	Body        string
	IsDraft     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Attachments []Attachment
}

type Recipient struct {
	UserID *int64 // nil — внешний отправитель
	Email  string
}

type EmailWithMetadata struct {
	Email
	IsRead     bool
	IsStarred  bool
	IsSpam     bool
	IsDeleted  bool
	ReceivedAt time.Time
	Recipients []string
}

// TODO: sender image is OPTIONAL (for external senders)
type EmailWithAvatar struct {
	Email
	SenderImagePath string
	Recipients      []string
}

type Draft struct {
	ID          int64
	SenderID    int64
	SenderEmail string
	Header      string
	Body        string
	Recipients  []string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Attachments []Attachment
}

type UserEmail struct {
	ID        int64
	EmailID   int64
	UserID    int64
	IsRead    bool
	IsDeleted bool
	IsStarred bool
	IsSpam    bool
	IsSender  bool
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

type Attachment struct {
	ID          int64
	EmailID     int64
	FileName    string
	ContentType string
	SizeBytes   int64
	StoragePath string
	CreatedAt   time.Time
}
