package service

import (
	"context"
	"io"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
)

// ─── input / output types ────────────────────────────────────────────────────

type UploadAttachmentInput struct {
	UserID      int64
	EmailID     int64
	Filename    string
	ContentType string
	Size        int64
	File        io.Reader
}

type DownloadAttachmentInput struct {
	UserID       int64
	EmailID      int64
	AttachmentID int64
}

type DeleteAttachmentsInput struct {
	UserID        int64
	EmailID       int64
	AttachmentIDs []int64
}

type GetAttachmentsInput struct {
	UserID  int64
	EmailID int64
}

type AttachmentResult struct {
	ID          int64
	EmailID     int64
	FileName    string
	ContentType string
	SizeBytes   int64
	StoragePath string
	CreatedAt   time.Time
}

type DownloadAttachmentResult struct {
	FileName    string
	ContentType string
	SizeBytes   int64
	Body        io.ReadCloser
}

type GetAttachmentsResult struct {
	Attachments []AttachmentResult
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func (s *Service) requireStorage() error {
	if s.storage == nil {
		return ErrStorageUnavailable
	}
	return nil
}

func attachmentToResult(a models.Attachment) AttachmentResult {
	return AttachmentResult{
		ID:          a.ID,
		EmailID:     a.EmailID,
		FileName:    a.FileName,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		StoragePath: a.StoragePath,
		CreatedAt:   a.CreatedAt,
	}
}

// ─── UploadAttachment ────────────────────────────────────────────────────────

// UploadAttachment streams a file into object storage and records its metadata
// in the database. The upload and the DB insert are kept consistent: if the
// DB insert fails the object is removed from storage.
func (s *Service) UploadAttachment(ctx context.Context, in UploadAttachmentInput) (*AttachmentResult, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}

	// Verify the caller has access to the email.
	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return nil, MapRepositoryError(err)
	}

	// Upload raw bytes to object storage.
	storageKey, err := s.storage.UploadAttachment(
		ctx, in.EmailID, in.Filename, in.File, in.Size, in.ContentType,
	)
	if err != nil {
		return nil, err
	}

	// Persist metadata in DB (transactional).
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		_ = s.storage.DeleteAttachment(ctx, storageKey)
		return nil, ErrTransaction
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	id, err := s.repo.InsertAttachment(ctx, tx, models.Attachment{
		EmailID:     in.EmailID,
		FileName:    in.Filename,
		ContentType: in.ContentType,
		SizeBytes:   in.Size,
		StoragePath: storageKey,
	})
	if err != nil {
		_ = s.storage.DeleteAttachment(ctx, storageKey)
		return nil, MapRepositoryError(err)
	}

	if err := tx.Commit(); err != nil {
		_ = s.storage.DeleteAttachment(ctx, storageKey)
		return nil, ErrTransaction
	}
	committed = true

	return &AttachmentResult{
		ID:          id,
		EmailID:     in.EmailID,
		FileName:    in.Filename,
		ContentType: in.ContentType,
		SizeBytes:   in.Size,
		StoragePath: storageKey,
		CreatedAt:   time.Now(),
	}, nil
}

// ─── DownloadAttachment ──────────────────────────────────────────────────────

// DownloadAttachment verifies access and streams the attachment body.
// The caller must close DownloadAttachmentResult.Body.
func (s *Service) DownloadAttachment(ctx context.Context, in DownloadAttachmentInput) (*DownloadAttachmentResult, error) {
	if err := s.requireStorage(); err != nil {
		return nil, err
	}

	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return nil, MapRepositoryError(err)
	}

	a, err := s.repo.GetAttachmentByID(ctx, in.AttachmentID, in.EmailID)
	if err != nil {
		return nil, MapRepositoryError(err)
	}
	if a == nil {
		return nil, ErrAttachmentNotFound
	}

	body, err := s.storage.DownloadAttachment(ctx, a.StoragePath)
	if err != nil {
		return nil, err
	}

	return &DownloadAttachmentResult{
		FileName:    a.FileName,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		Body:        body,
	}, nil
}

// ─── DeleteAttachments ───────────────────────────────────────────────────────

// DeleteAttachments removes DB records first (transactional), then best-effort
// deletes objects from storage so the DB remains consistent even if storage
// cleanup partially fails.
func (s *Service) DeleteAttachments(ctx context.Context, in DeleteAttachmentsInput) error {
	if err := s.requireStorage(); err != nil {
		return err
	}
	if len(in.AttachmentIDs) == 0 {
		return ErrEmptyIDs
	}

	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return MapRepositoryError(err)
	}

	// Resolve storage keys before deleting from DB.
	all, err := s.repo.GetAttachmentsByEmailIDs(ctx, []int64{in.EmailID})
	if err != nil {
		return MapRepositoryError(err)
	}

	keyByID := make(map[int64]string, len(all))
	for _, a := range all {
		keyByID[a.ID] = a.StoragePath
	}

	storageKeys := make([]string, 0, len(in.AttachmentIDs))
	for _, id := range in.AttachmentIDs {
		key, ok := keyByID[id]
		if !ok {
			return ErrAttachmentNotFound
		}
		storageKeys = append(storageKeys, key)
	}

	// Delete from DB.
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

	if err := s.repo.DeleteAttachments(ctx, tx, in.AttachmentIDs); err != nil {
		return MapRepositoryError(err)
	}
	if err := tx.Commit(); err != nil {
		return ErrTransaction
	}
	committed = true

	// Best-effort cleanup from storage.
	for _, key := range storageKeys {
		_ = s.storage.DeleteAttachment(ctx, key)
	}

	return nil
}

// ─── GetAttachments ──────────────────────────────────────────────────────────

// GetAttachments returns all attachment metadata for the given email.
func (s *Service) GetAttachments(ctx context.Context, in GetAttachmentsInput) (*GetAttachmentsResult, error) {
	if err := s.repo.CheckEmailAccess(ctx, in.UserID, in.EmailID); err != nil {
		return nil, MapRepositoryError(err)
	}

	attachments, err := s.repo.GetAttachmentsByEmailIDs(ctx, []int64{in.EmailID})
	if err != nil {
		return nil, MapRepositoryError(err)
	}

	result := &GetAttachmentsResult{
		Attachments: make([]AttachmentResult, 0, len(attachments)),
	}
	for _, a := range attachments {
		result.Attachments = append(result.Attachments, attachmentToResult(a))
	}

	return result, nil
}
