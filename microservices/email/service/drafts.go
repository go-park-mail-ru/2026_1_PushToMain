package service

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

type DraftResult struct {
	ID         int64
	SenderID   int64
	Header     string
	Body       string
	Recipients []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateDraftInput struct {
	UserID    int64
	Header    string
	Body      string
	Receivers []string
}

type UpdateDraftInput struct {
	UserID    int64
	DraftID   int64
	Header    string
	Body      string
	Receivers []string
}

type GetDraftInput struct {
	UserID  int64
	DraftID int64
}

type GetDraftsInput struct {
	UserID int64
	Limit  int
	Offset int
}

type GetDraftsResult struct {
	Drafts []DraftResult
	Limit  int
	Offset int
	Total  int
}

type DeleteDraftsInput struct {
	UserID   int64
	DraftIDs []int64
}

type SendDraftInput struct {
	UserID  int64
	DraftID int64
}

func validateDraftPayload(header, body string, receivers []string) error {
	if header == "" && body == "" && len(receivers) == 0 {
		return ErrDraftValidation
	}
	return nil
}

func draftToResult(d *models.Draft) *DraftResult {
	return &DraftResult{
		ID:         d.ID,
		SenderID:   d.SenderID,
		Header:     d.Header,
		Body:       d.Body,
		Recipients: d.Recipients,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func (s *Service) CreateDraft(ctx context.Context, in CreateDraftInput) (*DraftResult, error) {
	if err := validateDraftPayload(in.Header, in.Body, in.Receivers); err != nil {
		return nil, err
	}
	if s.drafts.MaxPerUser > 0 {
		count, err := s.repo.CountDrafts(ctx, in.UserID)
		if err != nil {
			return nil, MapRepositoryError(err)
		}
		if count >= s.drafts.MaxPerUser {
			return nil, ErrDraftsLimit
		}
	}
	draft, err := s.repo.CreateDraft(ctx, models.Draft{
		SenderID:   in.UserID,
		Header:     in.Header,
		Body:       in.Body,
		Recipients: in.Receivers,
	})
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(draft), nil
}

func (s *Service) UpdateDraft(ctx context.Context, in UpdateDraftInput) (*DraftResult, error) {
	if err := validateDraftPayload(in.Header, in.Body, in.Receivers); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDraft(ctx, in.UserID, models.Draft{
		ID:         in.DraftID,
		SenderID:   in.UserID,
		Header:     in.Header,
		Body:       in.Body,
		Recipients: in.Receivers,
	}); err != nil {
		return nil, MapRepositoryError(err)
	}
	saved, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(saved), nil
}

func (s *Service) GetDraftByID(ctx context.Context, in GetDraftInput) (*DraftResult, error) {
	d, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	return draftToResult(d), nil
}

func (s *Service) GetDrafts(ctx context.Context, in GetDraftsInput) (*GetDraftsResult, error) {
	drafts, err := s.repo.GetDrafts(ctx, in.UserID, in.Limit, in.Offset)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	total, err := s.repo.CountDrafts(ctx, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	out := make([]DraftResult, len(drafts))
	for i := range drafts {
		out[i] = *draftToResult(&drafts[i])
	}
	return &GetDraftsResult{
		Drafts: out, Limit: in.Limit, Offset: in.Offset, Total: total,
	}, nil
}

func (s *Service) DeleteDrafts(ctx context.Context, in DeleteDraftsInput) error {
	if len(in.DraftIDs) == 0 {
		return ErrEmptyIDs
	}
	if err := s.repo.DeleteDraftsBatch(ctx, in.UserID, in.DraftIDs); err != nil {
		return MapRepositoryError(err)
	}
	return nil
}

func (s *Service) SendDraft(ctx context.Context, in SendDraftInput) (*SendEmailResult, error) {
	d, err := s.repo.GetDraftByID(ctx, in.DraftID, in.UserID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if d.Header == "" || d.Body == "" || len(d.Recipients) == 0 {
		return nil, ErrDraftNotReady
	}
	recipients, err := s.resolveRecipients(ctx, d.Recipients)
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

	if err = s.repo.MarkDraftAsSentTx(ctx, tx, in.DraftID, in.UserID); err != nil {
		return nil, MapRepositoryError(err)
	}
	if err = s.repo.InsertEmailRecipients(ctx, tx, in.DraftID, recipients); err != nil {
		return nil, MapRepositoryError(err)
	}
	for _, r := range recipients {
		if r.UserID == nil {
			continue
		}
		if err = s.repo.InsertUserEmail(ctx, tx, *r.UserID, in.DraftID, false, true); err != nil {
			return nil, MapRepositoryError(err)
		}
	}
	if err = s.repo.InsertUserEmail(ctx, tx, in.UserID, in.DraftID, true, false); err != nil {
		return nil, MapRepositoryError(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, ErrTransaction
	}
	committed = true

	return &SendEmailResult{
		ID: in.DraftID, SenderID: in.UserID,
		Header: d.Header, Body: d.Body, CreatedAt: d.CreatedAt,
	}, nil
}
