package handler

import "time"

//easyjson:json
type SendEmailRequest struct {
	Header      string   `json:"header"`
	Body        string   `json:"body"`
	Receivers   []string `json:"receivers"`
	IsAnonymous bool     `json:"is_anonymous"`
}

//easyjson:json
type SendEmailResponse struct {
	ID            int64     `json:"email_id"`
	SenderID      int64     `json:"from"`
	Header        string    `json:"header"`
	Body          string    `json:"body"`
	IsAnonymous   bool      `json:"is_anonymous"`
	ParentEmailID *int64    `json:"parent_email_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

//easyjson:json
type ForwardEmailRequest struct {
	EmailID   int64    `json:"email_id"`
	Receivers []string `json:"receivers"`
}

//easyjson:json
type ReplyRequest struct {
	Header      string `json:"header"`
	Body        string `json:"body"`
	IsAnonymous bool   `json:"is_anonymous"`
}

//easyjson:json
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
	IsStarred     bool      `json:"is_starred"`
	IsAnonymous   bool      `json:"is_anonymous"`
	ParentEmailID *int64    `json:"parent_email_id,omitempty"`
}

//easyjson:json
type GetEmailsResponse struct {
	Emails      []EmailResponse `json:"emails"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
	Total       int             `json:"total"`
	UnreadCount int             `json:"unread_count"`
}

//easyjson:json
type MyEmailResponse struct {
	ID              int64     `json:"id"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	IsRead          bool      `json:"is_read"`
	IsStarred       bool      `json:"is_starred"`
	IsAnonymous     bool      `json:"is_anonymous"`
	ParentEmailID   *int64    `json:"parent_email_id,omitempty"`
	ReceiversEmails []string  `json:"receivers_emails"`
}

//easyjson:json
type GetMyEmailsResponse struct {
	Emails []MyEmailResponse `json:"emails"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

//easyjson:json
type GetEmailResponse struct {
	ID              int64     `json:"id"`
	SenderEmail     string    `json:"sender_email"`
	SenderName      string    `json:"sender_name"`
	SenderSurname   string    `json:"sender_surname"`
	Header          string    `json:"header"`
	Body            string    `json:"body"`
	CreatedAt       time.Time `json:"created_at"`
	SenderImagePath string    `json:"sender_image_path"`
	ReceiverList    []string  `json:"receiver_list"`
	IsAnonymous     bool      `json:"is_anonymous"`
	ParentEmailID   *int64    `json:"parent_email_id,omitempty"`
}

//easyjson:json
type MarkEmailsAsReadRequest struct {
	EmailIDs []int64 `json:"email_ids"`
}

//easyjson:json
type IDsRequest struct {
	IDs []int64 `json:"ids"`
}

//easyjson:json
type CreateDraftRequest struct {
	Header      string   `json:"header"`
	Body        string   `json:"body"`
	Receivers   []string `json:"receivers"`
	IsAnonymous bool     `json:"is_anonymous"`
}

//easyjson:json
type UpdateDraftRequest struct {
	Header      string   `json:"header"`
	Body        string   `json:"body"`
	Receivers   []string `json:"receivers"`
	IsAnonymous bool     `json:"is_anonymous"`
}

//easyjson:json
type DraftResponse struct {
	ID          int64     `json:"id"`
	SenderID    int64     `json:"sender_id"`
	Header      string    `json:"header"`
	Body        string    `json:"body"`
	Receivers   []string  `json:"receivers"`
	IsAnonymous bool      `json:"is_anonymous"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

//easyjson:json
type GetDraftsResponse struct {
	Drafts []DraftResponse `json:"drafts"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Total  int             `json:"total"`
}

//easyjson:json
type AnonymousRejectedResponse struct {
	Error          string   `json:"error"`
	RejectedEmails []string `json:"rejected_emails"`
	DraftID        int64    `json:"draft_id"`
}

//easyjson:json
type AttachmentResponse struct {
	ID          int64     `json:"id"`
	EmailID     int64     `json:"email_id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

//easyjson:json
type GetAttachmentsResponse struct {
	Attachments []AttachmentResponse `json:"attachments"`
}
