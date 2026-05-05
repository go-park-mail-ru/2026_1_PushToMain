package service

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

type GetEmailsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetEmailsResult struct {
	Emails      []EmailResult
	Limit       int
	Offset      int
	Total       int
	UnreadCount int
}

type EmailResult struct {
	ID            int64
	SenderID      int64
	SenderEmail   string
	SenderName    string
	SenderSurname string
	ReceiverList  []string
	Header        string
	Body          string
	CreatedAt     time.Time
	IsRead        bool
	IsStarred     bool
}

type GetMyEmailsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetMyEmailsResult struct {
	Emails []MyEmailResult
	Limit  int
	Offset int
	Total  int
}

type MyEmailResult struct {
	ID              int64
	SenderID        int64
	Header          string
	Body            string
	CreatedAt       time.Time
	IsRead          bool
	ReceiversEmails []string
}

func (s *Service) GetEmailsByReceiver(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetEmailsByReceiver(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetEmailsCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	unread, err := s.repo.GetUnreadEmailsCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, total, unread)
}

func (s *Service) GetEmailsBySender(ctx context.Context, in GetMyEmailsInput) (*GetMyEmailsResult, error) {
	emails, err := s.repo.GetEmailsBySender(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.GetSenderEmailsCount(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	out := make([]MyEmailResult, len(emails))
	for i, em := range emails {
		out[i] = MyEmailResult{
			ID:              em.ID,
			SenderID:        em.SenderID,
			Header:          em.Header,
			Body:            em.Body,
			CreatedAt:       em.CreatedAt,
			IsRead:          em.IsRead,
			ReceiversEmails: em.ReceiversEmails,
		}
	}
	return &GetMyEmailsResult{
		Emails: out,
		Limit:  in.Limit,
		Offset: in.Offset,
		Total:  total,
	}, nil
}

func (s *Service) buildEmailsResult(
	ctx context.Context, emails []models.EmailWithMetadata,
	limit, offset, total, unread int,
) (*GetEmailsResult, error) {
	out := make([]EmailResult, len(emails))
	for i, em := range emails {
		user, err := s.userClient.GetUserByID(ctx, em.SenderID)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
		out[i] = EmailResult{
			ID:            em.ID,
			SenderID:      em.SenderID,
			SenderEmail:   user.Email,
			SenderName:    user.Name,
			SenderSurname: user.Surname,
			ReceiverList:  em.ReceiversEmails,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsRead:        em.IsRead,
			IsStarred:     em.IsStarred,
		}
	}
	return &GetEmailsResult{
		Emails: out, Limit: limit, Offset: offset,
		Total: total, UnreadCount: unread,
	}, nil
}

type SendEmailInput struct {
	UserId    int64
	Header    string
	Body      string
	Receivers []string
}

type SendEmailResult struct {
	ID        int64
	SenderID  int64
	Header    string
	Body      string
	CreatedAt time.Time
}

func (s *Service) SendEmail(ctx context.Context, in SendEmailInput) (*SendEmailResult, error) {
	receiverIDs, err := s.ResolveReceivers(ctx, in.Receivers)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	email := models.Email{SenderID: in.UserId, Header: in.Header, Body: in.Body}
	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	email.ID = emailID

	for _, rid := range receiverIDs {
		if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, rid, false); err != nil {
			return nil, MapRepositoryError(err)
		}
	}
	if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, email.SenderID, true); err != nil {
		return nil, MapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	return &SendEmailResult{
		ID: email.ID, SenderID: email.SenderID,
		Header: email.Header, Body: email.Body, CreatedAt: email.CreatedAt,
	}, nil
}

type ForwardEmailInput struct {
	UserID    int64
	EmailID   int64
	Receivers []string
}

func (s *Service) ForwardEmail(ctx context.Context, in ForwardEmailInput) error {
	src, err := s.GetEmailByID(ctx, GetEmailInput{UserID: in.UserID, EmailID: in.EmailID})
	if err != nil {
		return MapRepositoryError(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	email := models.Email{SenderID: in.UserID, Header: src.Header, Body: src.Body}
	emailID, err := s.repo.SaveEmailWithTx(ctx, tx, email)
	if err != nil {
		return MapRepositoryError(err)
	}

	receiverIDs, err := s.ResolveReceivers(ctx, in.Receivers)
	if err != nil {
		return MapRepositoryError(err)
	}
	for _, rid := range receiverIDs {
		if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, rid, false); err != nil {
			return MapRepositoryError(err)
		}
	}
	if err = s.repo.AddEmailUserWithTx(ctx, tx, emailID, in.UserID, true); err != nil {
		return MapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return ErrTransaction
	}
	committed = true
	return nil
}

type GetEmailInput struct {
	UserID  int64
	EmailID int64
}

type GetEmailResult struct {
	ID              int64
	SenderID        int64
	SenderEmail     string
	SenderName      string
	SenderSurname   string
	Header          string
	Body            string
	CreatedAt       time.Time
	SenderImagePath string
	ReceiverList    []string
}

func (s *Service) GetEmailByID(ctx context.Context, in GetEmailInput) (*GetEmailResult, error) {
	if err := s.repo.CheckEmailAccess(ctx, in.EmailID, in.UserID); err != nil {
		return nil, MapRepositoryError(err)
	}
	em, err := s.repo.GetEmailByID(ctx, in.EmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	user, err := s.userClient.GetUserByID(ctx, em.SenderID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return &GetEmailResult{
		ID:              em.ID,
		SenderID:        em.SenderID,
		SenderEmail:     user.Email,
		SenderName:      user.Name,
		SenderSurname:   user.Surname,
		Header:          em.Header,
		Body:            em.Body,
		CreatedAt:       em.CreatedAt,
		SenderImagePath: em.SenderImagePath,
		ReceiverList:    em.ReceiversEmails,
	}, nil
}

type GetEmailsByIDsResult struct {
	Emails      []EmailResult
	UnreadCount int
}

func (s *Service) GetEmailsByIDs(
	ctx context.Context,
	emailIDs []int64,
	userID int64,
) (*GetEmailsByIDsResult, error) {

	if len(emailIDs) == 0 {
		return &GetEmailsByIDsResult{
			Emails:      []EmailResult{},
			UnreadCount: 0,
		}, nil
	}

	emails, err := s.repo.GetEmailsByIDs(ctx, emailIDs, userID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	// считаем unread вручную
	unread := 0
	for _, e := range emails {
		if !e.IsRead {
			unread++
		}
	}

	// build response (как у buildEmailsResult)
	out := make([]EmailResult, len(emails))

	for i, em := range emails {
		user, err := s.userClient.GetUserByID(ctx, em.SenderID)
		if err != nil {
			return nil, MapRepositoryError(err)
		}

		out[i] = EmailResult{
			ID:            em.ID,
			SenderID:      em.SenderID,
			SenderEmail:   user.Email,
			SenderName:    user.Name,
			SenderSurname: user.Surname,
			ReceiverList:  em.ReceiversEmails,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsRead:        em.IsRead,
			IsStarred:     em.IsStarred,
		}
	}

	return &GetEmailsByIDsResult{
		Emails:      out,
		UnreadCount: unread,
	}, nil
}

func (s *Service) CheckEmailAccess(ctx context.Context, in GetEmailInput) error {
	return s.repo.CheckEmailAccess(ctx, in.EmailID, in.UserID)
}

func (s *Service) ResolveReceivers(ctx context.Context, receiverEmails []string) ([]int64, error) {
	if len(receiverEmails) == 0 {
		return nil, ErrNoValidReceivers
	}
	users, err := s.repo.GetUsersByEmails(ctx, receiverEmails)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if len(users) == 0 {
		return nil, ErrNoValidReceivers
	}
	out := make([]int64, len(users))
	for i, u := range users {
		out[i] = u.ID
	}
	return out, nil
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID []int64
}

func (s *Service) MarkEmailAsRead(ctx context.Context, in MarkAsReadInput) error {
	for _, id := range in.EmailID {
		if err := s.repo.MarkEmailAsRead(ctx, id, in.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}

func (s *Service) MarkEmailAsUnRead(ctx context.Context, in MarkAsReadInput) error {
	for _, id := range in.EmailID {
		if err := s.repo.MarkEmailAsUnRead(ctx, id, in.UserID); err != nil {
			return MapRepositoryError(err)
		}
	}
	return nil
}
