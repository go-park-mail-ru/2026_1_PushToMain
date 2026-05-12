package handler

import "time"

type SendEmailRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type SendEmailResponse struct {
	ID        int64     `json:"email_id"`
	SenderID  int64     `json:"from"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type ForwardEmailRequest struct {
	EmailID   int64    `json:"email_id"`
	Receivers []string `json:"receivers"`
}

type EmailResponse struct {
	ID            int64     `json:"id"`
	SenderEmail   string    `json:"sender_email"`
	SenderName    string    `json:"sender_name"`
	SenderSurname string    `json:"sender_surname"`
	ReceiverList  []string  `json:"receiver_list"`
	Header        string    `json:"header"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	IsRead        bool      `json:"is_read"`
}

type GetEmailsResponse struct {
	Emails      []EmailResponse `json:"emails"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
	Total       int             `json:"total"`
	UnreadCount int             `json:"unread_count"`
}

type MyEmailResponse struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	IsRead          bool      `json:"is_read"`
	ReceiversEmails []string  `json:"receivers_emails"`
}

type GetMyEmailsResponse struct {
	Emails []MyEmailResponse `json:"emails"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

type GetEmailResponse struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	SenderEmail     string    `json:"sender_email"`
	SenderName      string    `json:"sender_name"`
	SenderSurname   string    `json:"sender_surname"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	SenderImagePath string    `json:"sender_image_path"`
	ReceiverList    []string  `json:"receiver_list"`
}

type MarkEmailsAsReadRequest struct {
	EmailIDs []int64 `json:"email_ids"`
}

type IDsRequest struct {
	IDs []int64 `json:"ids"`
}

type CreateDraftRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type UpdateDraftRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type DraftResponse struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	Receivers []string  `json:"receivers"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetDraftsResponse struct {
	Drafts []DraftResponse `json:"drafts"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Total  int             `json:"total"`
}
