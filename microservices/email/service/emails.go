package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/lmtp"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/smtp"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"
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
	SenderID      *int64
	SenderEmail   string
	SenderName    string
	SenderSurname string
	ReceiverList  []string
	Header        string
	Body          string
	CreatedAt     time.Time
	IsRead        bool
	IsStarred     bool
	IsAnonymous   bool
	ParentEmailID *int64
}

type GetMyEmailsResult struct {
	Emails []MyEmailResult
	Limit  int
	Offset int
	Total  int
}

type MyEmailResult struct {
	ID              int64
	SenderID        *int64
	Header          string
	Body            string
	CreatedAt       time.Time
	IsRead          bool
	IsStarred       bool
	IsAnonymous     bool
	ParentEmailID   *int64
	ReceiversEmails []string
}

type GetEmailInput struct {
	UserID  int64
	EmailID int64
}

type GetEmailResult struct {
	ID              int64
	SenderID        *int64
	SenderEmail     string
	SenderName      string
	SenderSurname   string
	Header          string
	Body            string
	IsStarred       bool
	IsAnonymous     bool
	ParentEmailID   *int64
	CreatedAt       time.Time
	SenderImagePath string
	ReceiverList    []string
}

type SendEmailInput struct {
	UserId        int64
	Header        string
	Body          string
	Receivers     []string
	IsAnonymous   bool
	ParentEmailID *int64

	Files       []multipart.File
	FileHeaders []*multipart.FileHeader
}

type SendEmailResult struct {
	ID          int64
	SenderID    int64
	Header      string
	Body        string
	IsAnonymous bool
	CreatedAt   time.Time
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

type SupportSender struct {
	UserID  *int64
	Email   string
	Name    string
	Surname string
}

type GetEmailForSupportResult struct {
	ID          int64
	Sender      SupportSender
	Recipients  []string
	Header      string
	Body        string
	CreatedAt   time.Time
	IsAnonymous bool
	IsDraft     bool
	Attachments []AttachmentResult
}

func (s *Service) SendSystemEmail(
	ctx context.Context,
	recipientUserId int64,
	recipientEmail string,
	systemEmail string,
	header string,
	body string,
) error {
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

	emailId, err := s.repo.InsertExternalEmail(ctx, tx, systemEmail, header, body)
	if err != nil {
		return MapRepositoryError(err)
	}

	recipients := []models.Recipient{{Email: recipientEmail, UserID: &recipientUserId}}
	if err = s.repo.InsertEmailRecipients(ctx, tx, emailId, recipients); err != nil {
		return MapRepositoryError(err)
	}

	if err = s.repo.InsertUserEmail(ctx, tx, recipientUserId, emailId, false, true); err != nil {
		return MapRepositoryError(err)
	}

	if err = tx.Commit(); err != nil {
		return ErrTransaction
	}
	committed = true
	return nil
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
	return s.buildEmailsResult(ctx, in.UserID, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
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
	return s.buildEmailsResult(ctx, in.UserID, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
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
	return s.buildEmailsResult(ctx, in.UserID, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
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
	return s.buildEmailsResult(ctx, in.UserID, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
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
	return s.buildEmailsResult(ctx, in.UserID, emails, in.Limit, in.Offset, stats.Total, stats.Unread)
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
			IsAnonymous:     em.IsAnonymous,
			ParentEmailID:   em.ParentEmailID,
			ReceiversEmails: em.Recipients,
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
	ctx context.Context,
	viewerID int64,
	emails []models.EmailWithMetadata,
	limit, offset, total, unread int,
) (*GetEmailsResult, error) {
	out := make([]EmailResult, len(emails))
	for i, em := range emails {
		hide := em.IsAnonymous && (em.SenderID == nil || *em.SenderID != viewerID)

		var senderID *int64
		var senderEmail, senderName, senderSurname string

		if hide {
			// ...
		} else if em.SenderID != nil {
			user, err := s.userClient.GetUserByID(ctx, *em.SenderID)
			senderID = em.SenderID
			if err != nil {
				senderEmail = em.SenderEmail
			} else {
				senderEmail = user.Email
				senderName = user.Name
				senderSurname = user.Surname
			}
		} else {
			senderEmail = em.SenderEmail
		}

		out[i] = EmailResult{
			ID:            em.ID,
			SenderID:      senderID,
			SenderEmail:   senderEmail,
			SenderName:    senderName,
			SenderSurname: senderSurname,
			ReceiverList:  em.Recipients,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsRead:        em.IsRead,
			IsStarred:     em.IsStarred,
			IsAnonymous:   em.IsAnonymous,
			ParentEmailID: em.ParentEmailID,
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

	result := &GetEmailResult{
		ID:            em.ID,
		Header:        em.Header,
		Body:          em.Body,
		CreatedAt:     em.CreatedAt,
		ReceiverList:  em.Recipients,
		IsAnonymous:   em.IsAnonymous,
		ParentEmailID: em.ParentEmailID,
	}

	hide := em.IsAnonymous && (em.SenderID == nil || *em.SenderID != in.UserID)
	if !hide {
		result.SenderID = em.SenderID
		result.SenderEmail = em.SenderEmail
		result.SenderImagePath = em.SenderImagePath
		if em.SenderID != nil {
			user, err := s.userClient.GetUserByID(ctx, *em.SenderID)
			if err != nil {
				return nil, MapRepositoryError(err)
			}
			result.SenderEmail = user.Email
			result.SenderName = user.Name
			result.SenderSurname = user.Surname
		}
	}

	return result, nil
}

func (s *Service) GetEmailIdsByUserEmailIds(ctx context.Context, userEmailIDs []int64) ([]int64, error) {
	result, err := s.repo.GetEmailIdsByUserEmailIds(ctx, userEmailIDs)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return result, nil
}

func (s *Service) GetUserEmailID(ctx context.Context, emailID int64, userID int64) (int64, error) {
	result, err := s.repo.GetUserEmailID(ctx, emailID, userID)
	if err != nil {
		return 0, MapRepositoryError(err)
	}
	return result, nil
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
		email := EmailResult{
			ID:            em.ID,
			ReceiverList:  em.Recipients,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsAnonymous:   em.IsAnonymous,
			ParentEmailID: em.ParentEmailID,
		}

		hide := em.IsAnonymous && (em.SenderID == nil || *em.SenderID != userID)
		if !hide && em.SenderID != nil {
			senderUser, err := s.userClient.GetUserByID(ctx, *em.SenderID)
			if err != nil {
				return nil, MapRepositoryError(err)
			}
			email.SenderID = em.SenderID
			email.SenderEmail = senderUser.Email
			email.SenderName = senderUser.Name
			email.SenderSurname = senderUser.Surname
		}

		out = append(out, email)
	}
	return &GetEmailsByIDsResult{Emails: out, UnreadCount: 0}, nil
}

func (s *Service) CheckEmailAccess(ctx context.Context, in GetEmailInput) error {
	return s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID)
}

func (s *Service) SendEmail(ctx context.Context, in SendEmailInput) (*SendEmailResult, error) {
	if in.IsAnonymous {
		return s.sendAnonymousEmail(ctx, in)
	}
	recipients, err := s.resolveRecipients(ctx, in.Receivers)
	if err != nil {
		return nil, err
	}
	return s.sendEmailTx(ctx, in.UserId, in.Header, in.Body, recipients, in.Files, in.FileHeaders, false, in.ParentEmailID)
}
func (s *Service) sendAnonymousEmail(ctx context.Context, in SendEmailInput) (*SendEmailResult, error) {
	recipients, usersByEmail, err := s.resolveRecipientsWithUsers(ctx, in.Receivers)
	if err != nil {
		return nil, err
	}

	for _, r := range recipients {
		if r.UserID == nil {
			return nil, ErrAnonymousExternal
		}
	}

	var rejected []string
	for _, r := range recipients {
		u := usersByEmail[r.Email]
		if u == nil || !u.AcceptAnonymous {
			rejected = append(rejected, r.Email)
		}
	}

	if len(rejected) > 0 {
		payloads, err := readFilePayloads(in.Files, in.FileHeaders)
		if err != nil {
			return nil, err
		}
		draftID, err := s.saveDraftWithPayloads(ctx, in.UserId, in.Header, in.Body, recipients, true, payloads)
		if err != nil {
			return nil, err
		}
		return nil, &ErrAnonymousRejected{Emails: rejected, DraftID: draftID}
	}

	return s.sendEmailTx(ctx, in.UserId, in.Header, in.Body, recipients, in.Files, in.FileHeaders, true, in.ParentEmailID)
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
	_, err = s.sendEmailTx(ctx, in.UserID, src.Header, src.Body, recipients, nil, nil, false, nil)
	return err
}

type filePayload struct {
	data        []byte
	filename    string
	contentType string
	size        int64
}

func readFilePayloads(files []multipart.File, fileHeaders []*multipart.FileHeader) ([]filePayload, error) {
	payloads := make([]filePayload, 0, len(files))
	for i, f := range files {
		if i >= len(fileHeaders) {
			break
		}
		fh := fileHeaders[i]
		data, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		payloads = append(payloads, filePayload{
			data:        data,
			filename:    fh.Filename,
			contentType: ct,
			size:        fh.Size,
		})
	}
	return payloads, nil
}

func (s *Service) sendEmailTx(
	ctx context.Context,
	senderID int64,
	header, body string,
	recipients []models.Recipient,
	files []multipart.File,
	fileHeaders []*multipart.FileHeader,
	isAnonymous bool,
	parentEmailID *int64,
) (*SendEmailResult, error) {
	sender, err := s.userClient.GetUserByID(ctx, senderID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	payloads, err := readFilePayloads(files, fileHeaders)
	if err != nil {
		return nil, err
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
		SenderID:      &senderID,
		SenderEmail:   sender.Email,
		Header:        header,
		Body:          body,
		IsDraft:       false,
		IsAnonymous:   isAnonymous,
		ParentEmailID: parentEmailID,
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
		if err = s.repo.InsertUserEmail(ctx, tx, *r.UserID, emailID, false, true); err != nil {
			return nil, MapRepositoryError(err)
		}
	}

	if err = s.repo.InsertUserEmail(ctx, tx, senderID, emailID, true, false); err != nil {
		return nil, MapRepositoryError(err)
	}

	if len(payloads) > 0 && s.storage != nil {
		for _, p := range payloads {
			storageKey, uploadErr := s.storage.UploadAttachment(
				ctx, emailID, p.filename, bytes.NewReader(p.data), p.size, p.contentType,
			)
			if uploadErr != nil {
				return nil, uploadErr
			}
			if _, insertErr := s.repo.InsertAttachment(ctx, tx, models.Attachment{
				EmailID:     emailID,
				FileName:    p.filename,
				ContentType: p.contentType,
				SizeBytes:   p.size,
				StoragePath: storageKey,
			}); insertErr != nil {
				_ = s.storage.DeleteAttachment(ctx, storageKey)
				return nil, MapRepositoryError(insertErr)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	if !isAnonymous && s.smtpClient != nil {
		external := collectExternal(recipients)
		if len(external) > 0 {
			smtpAttachments := make([]smtp.Attachment, 0, len(payloads))
			for _, p := range payloads {
				smtpAttachments = append(smtpAttachments, smtp.Attachment{
					Filename: p.filename,
					Data:     p.data,
					MIMEType: p.contentType,
				})
			}
			if err := s.smtpClient.SendEmail(sender.Name, sender.Surname, sender.Email, external, header, body, smtpAttachments); err != nil {
				// TODO: надо сделать гарантированную доставку
				return nil, fmt.Errorf("smtp send: %w", err)
			}
		}
	}

	return &SendEmailResult{
		ID:          emailID,
		SenderID:    senderID,
		Header:      header,
		Body:        body,
		IsAnonymous: isAnonymous,
		CreatedAt:   time.Now(),
	}, nil
}

func (s *Service) saveDraftWithPayloads(
	ctx context.Context,
	senderID int64,
	header, body string,
	recipients []models.Recipient,
	isAnonymous bool,
	payloads []filePayload,
) (int64, error) {
	sender, err := s.userClient.GetUserByID(ctx, senderID)
	if err != nil {
		return 0, MapRepositoryError(err)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	emailID, err := s.repo.InsertEmail(ctx, tx, models.Email{
		SenderID:    &senderID,
		SenderEmail: sender.Email,
		Header:      header,
		Body:        body,
		IsDraft:     true,
		IsAnonymous: isAnonymous,
	})
	if err != nil {
		return 0, MapRepositoryError(err)
	}

	if err := s.repo.InsertEmailRecipients(ctx, tx, emailID, recipients); err != nil {
		return 0, MapRepositoryError(err)
	}

	var uploadedKeys []string
	if len(payloads) > 0 && s.storage != nil {
		for _, p := range payloads {
			key, err := s.storage.UploadAttachment(
				ctx, emailID, p.filename, bytes.NewReader(p.data), p.size, p.contentType,
			)
			if err != nil {
				for _, k := range uploadedKeys {
					_ = s.storage.DeleteAttachment(ctx, k)
				}
				return 0, err
			}
			uploadedKeys = append(uploadedKeys, key)
			if _, err := s.repo.InsertAttachment(ctx, tx, models.Attachment{
				EmailID: emailID, FileName: p.filename, ContentType: p.contentType,
				SizeBytes: p.size, StoragePath: key,
			}); err != nil {
				for _, k := range uploadedKeys {
					_ = s.storage.DeleteAttachment(ctx, k)
				}
				return 0, MapRepositoryError(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		for _, k := range uploadedKeys {
			_ = s.storage.DeleteAttachment(ctx, k)
		}
		return 0, ErrTransaction
	}
	committed = true
	return emailID, nil
}

func (s *Service) ReceiveExternalEmail(ctx context.Context, from string, to []string, subject string, parsed lmtp.ParsedEmail) error {
	users, err := s.userClient.GetUsersByEmails(ctx, to)
	if err != nil {
		return MapRepositoryError(err)
	}
	byEmail := make(map[string]int64, len(users))
	for _, u := range users {
		byEmail[u.Email] = u.Id
	}

	recipients := make([]models.Recipient, 0, len(to))
	for _, e := range to {
		rec := models.Recipient{Email: e}
		if id, ok := byEmail[e]; ok {
			id := id
			rec.UserID = &id
		}
		recipients = append(recipients, rec)
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

	emailID, err := s.repo.InsertExternalEmail(ctx, tx, from, subject, parsed.Body)
	if err != nil {
		return MapRepositoryError(err)
	}
	if err = s.repo.InsertEmailRecipients(ctx, tx, emailID, recipients); err != nil {
		return MapRepositoryError(err)
	}
	for _, r := range recipients {
		if r.UserID == nil {
			continue
		}
		if err = s.repo.InsertUserEmail(ctx, tx, *r.UserID, emailID, false, true); err != nil {
			return MapRepositoryError(err)
		}
	}

	var uploadedKeys []string
	if len(parsed.Attachments) > 0 && s.storage != nil {
		for _, a := range parsed.Attachments {
			ct := a.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			key, err := s.storage.UploadAttachment(
				ctx, emailID, a.Filename,
				bytes.NewReader(a.Data), int64(len(a.Data)), ct,
			)
			if err != nil {
				for _, k := range uploadedKeys {
					_ = s.storage.DeleteAttachment(ctx, k)
				}
				return err
			}
			uploadedKeys = append(uploadedKeys, key)
			if _, err := s.repo.InsertAttachment(ctx, tx, models.Attachment{
				EmailID:     emailID,
				FileName:    a.Filename,
				ContentType: ct,
				SizeBytes:   int64(len(a.Data)),
				StoragePath: key,
			}); err != nil {
				for _, k := range uploadedKeys {
					_ = s.storage.DeleteAttachment(ctx, k)
				}
				return MapRepositoryError(err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		for _, k := range uploadedKeys {
			_ = s.storage.DeleteAttachment(ctx, k)
		}
		return ErrTransaction
	}
	committed = true
	return nil
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
	recipients, _, err := s.resolveRecipientsWithUsers(ctx, emails)
	return recipients, err
}

func (s *Service) resolveRecipientsWithUsers(
	ctx context.Context, emails []string,
) ([]models.Recipient, map[string]*userpb.User, error) {
	if len(emails) == 0 {
		return nil, nil, ErrNoValidReceivers
	}
	users, err := s.userClient.GetUsersByEmails(ctx, emails)
	if err != nil {
		return nil, nil, MapRepositoryError(err)
	}
	byEmail := make(map[string]*userpb.User, len(users))
	for _, u := range users {
		byEmail[u.Email] = u
	}

	out := make([]models.Recipient, 0, len(emails))
	for _, e := range emails {
		domain := extractDomain(e)
		rec := models.Recipient{Email: e}

		if u, ok := byEmail[e]; ok {
			id := u.Id
			rec.UserID = &id
		} else if isLocalDomain(domain) {
			return nil, nil, &ErrRecipientNotFound{Email: e}
		}
		out = append(out, rec)
	}

	return out, byEmail, nil
}

func (s *Service) GetEmailForSupport(ctx context.Context, emailID int64) (*GetEmailForSupportResult, error) {
	em, senderName, senderSurname, err := s.repo.GetEmailForSupport(ctx, emailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	attachments, err := s.repo.GetAttachmentsByEmailIDs(ctx, []int64{emailID})
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	attResults := make([]AttachmentResult, 0, len(attachments))
	for _, a := range attachments {
		attResults = append(attResults, attachmentToResult(a))
	}

	return &GetEmailForSupportResult{
		ID: em.ID,
		Sender: SupportSender{
			UserID:  em.SenderID,
			Email:   em.SenderEmail,
			Name:    senderName,
			Surname: senderSurname,
		},
		Recipients:  em.Recipients,
		Header:      em.Header,
		Body:        em.Body,
		CreatedAt:   em.CreatedAt,
		IsAnonymous: em.IsAnonymous,
		IsDraft:     em.IsDraft,
		Attachments: attResults,
	}, nil
}

func collectExternal(recipients []models.Recipient) []string {
	var out []string
	for _, r := range recipients {
		if r.UserID == nil {
			out = append(out, r.Email)
		}
	}
	return out
}

func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func isLocalDomain(domain string) bool {
	return domain == "e-smail.ru"
}
