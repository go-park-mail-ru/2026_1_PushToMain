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

type GetMyEmailsInput struct {
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
	IsStarred       bool
	ReceiversEmails []string
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
	IsStarred       bool
	CreatedAt       time.Time
	SenderImagePath string
	ReceiverList    []string
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

type ForwardEmailInput struct {
	UserID    int64
	EmailID   int64
	Receivers []string
}

type MarkAsReadInput struct {
	UserID  int64
	EmailID []int64
}

type GetEmailsByIDsResult struct {
	Emails      []EmailResult
	UnreadCount int
}

func (s *Service) GetEmailsByReceiver(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetInboxEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats, err := s.repo.GetInboxStats(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
}

func (s *Service) GetAllEmailsByUser(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetAllEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats, err := s.repo.GetReceivedStats(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	sentStat, err := s.repo.CountSentEmails(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats.Total += sentStat
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
}

func (s *Service) GetSpamEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetSpamEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats, err := s.repo.GetSpamStats(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
}

func (s *Service) GetTrashEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetTrashEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats, err := s.repo.GetTrashStats(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
}

func (s *Service) GetFavoriteEmails(ctx context.Context, in GetEmailsInput) (*GetEmailsResult, error) {
	emails, err := s.repo.GetFavoriteEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	stats, err := s.repo.GetFavoritesStats(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return s.buildEmailsResult(ctx, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
}

func (s *Service) GetEmailsBySender(ctx context.Context, in GetMyEmailsInput) (*GetMyEmailsResult, error) {
	emails, err := s.repo.GetSentEmails(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.CountSentEmails(ctx, in.UserID)
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
			IsStarred:       em.IsStarred,
			ReceiversEmails: em.Recipients,
		}
	}
	return &GetMyEmailsResult{
		Emails: out, Limit: in.Limit, Offset: in.Offset, Total: total,
	}, nil
}

func (s *Service) buildEmailsResult(
	ctx context.Context,
	emails []models.EmailWithMetadata,
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
			ReceiverList:  em.Recipients,
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

func (s *Service) GetEmailByID(ctx context.Context, in GetEmailInput) (*GetEmailResult, error) {
	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return nil, MapRepositoryError(err)
	}
	em, err := s.repo.GetEmailByID(ctx, in.EmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if em == nil {
		return nil, ErrEmailNotFound
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
		ReceiverList:    em.Recipients,
	}, nil
}

func (s *Service) GetEmailsByIDs(ctx context.Context, emailIDs []int64, userID int64) (*GetEmailsByIDsResult, error) {
	if len(emailIDs) == 0 {
		return &GetEmailsByIDsResult{Emails: []EmailResult{}, UnreadCount: 0}, nil
	}
	out := make([]EmailResult, 0, len(emailIDs))
	for _, id := range emailIDs {
		em, err := s.repo.GetEmailByID(ctx, id)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
		if em == nil {
			continue
		}
		user, err := s.userClient.GetUserByID(ctx, em.SenderID)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
		out = append(out, EmailResult{
			ID:            em.ID,
			SenderID:      em.SenderID,
			SenderEmail:   user.Email,
			SenderName:    user.Name,
			SenderSurname: user.Surname,
			ReceiverList:  em.Recipients,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
		})
	}
	return &GetEmailsByIDsResult{Emails: out, UnreadCount: 0}, nil
}

func (s *Service) CheckEmailAccess(ctx context.Context, in GetEmailInput) error {
	return s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID)
}

func (s *Service) SendEmail(ctx context.Context, in SendEmailInput) (*SendEmailResult, error) {
	recipients, err := s.resolveRecipients(ctx, in.Receivers)
	if err != nil {
		return nil, err
	}
	return s.sendEmailTx(ctx, in.UserId, in.Header, in.Body, recipients)
}

func (s *Service) ForwardEmail(ctx context.Context, in ForwardEmailInput) error {
	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return MapRepositoryError(err)
	}
	src, err := s.repo.GetEmailByID(ctx, in.EmailID)
	if err != nil {
		return MapRepositoryError(err)
	}
	if src == nil {
		return ErrEmailNotFound
	}
	recipients, err := s.resolveRecipients(ctx, in.Receivers)
	if err != nil {
		return err
	}
	_, err = s.sendEmailTx(ctx, in.UserID, src.Header, src.Body, recipients)
	return err
}

func (s *Service) sendEmailTx(
	ctx context.Context,
	senderID int64,
	header, body string,
	recipients []models.Recipient,
) (*SendEmailResult, error) {
	sender, err := s.userClient.GetUserByID(ctx, senderID)
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

	emailID, err := s.repo.InsertEmail(ctx, tx, models.Email{
		SenderID:    senderID,
		SenderEmail: sender.Email,
		Header:      header,
		Body:        body,
		IsDraft:     false,
	})
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if err = s.repo.InsertEmailRecipients(ctx, tx, emailID, recipients); err != nil {
		return nil, MapRepositoryError(err)
	}
	for _, r := range recipients {
		if r.UserID == nil {
			continue
		}
		if err = s.repo.InsertUserEmail(ctx, tx, *r.UserID, emailID, true); err != nil {
			return nil, MapRepositoryError(err)
		}
	}
	if err = s.repo.InsertUserEmail(ctx, tx, senderID, emailID, false); err != nil {
		return nil, MapRepositoryError(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	return &SendEmailResult{
		ID: emailID, SenderID: senderID,
		Header: header, Body: body, CreatedAt: time.Now(),
	}, nil
}

func (s *Service) MarkEmailAsRead(ctx context.Context, in MarkAsReadInput) error {
	if len(in.EmailID) == 0 {
		return ErrEmptyIDs
	}
	if err := s.repo.ReadEmails(ctx, in.UserID, in.EmailID); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) MarkEmailAsUnRead(ctx context.Context, in MarkAsReadInput) error {
	if len(in.EmailID) == 0 {
		return ErrEmptyIDs
	}
	if err := s.repo.UnreadEmails(ctx, in.UserID, in.EmailID); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) UnblockSenders(ctx context.Context, in BatchInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.repo.UnblockSendersBatch(ctx, in.UserID, in.EmailIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) resolveRecipients(ctx context.Context, emails []string) ([]models.Recipient, error) {
	if len(emails) == 0 {
		return nil, ErrNoValidReceivers
	}
	users, err := s.userClient.GetUsersByEmails(ctx, emails)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	byEmail := make(map[string]int64, len(users))
	for _, u := range users {
		byEmail[u.Email] = u.Id
	}
	out := make([]models.Recipient, 0, len(emails))
	hasInternal := false
	for _, e := range emails {
		rec := models.Recipient{Email: e}
		if id, ok := byEmail[e]; ok {
			rec.UserID = &id
			hasInternal = true
		}
		out = append(out, rec)
	}
	if !hasInternal {
		return nil, ErrNoValidReceivers
	}
	return out, nil
}
