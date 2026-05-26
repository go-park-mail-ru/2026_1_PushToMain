package service

import (
	"errors"
	"fmt"

	repository "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailNotFound      = errors.New("email not found")
	ErrNoValidReceivers   = errors.New("no valid receivers found")
	ErrAccessDenied       = errors.New("don't have access to this email")
	ErrTransaction        = errors.New("transaction failed")
	ErrConflict           = errors.New("conflict")
	ErrBadRequest         = errors.New("bad request")
	ErrEmptyIDs           = errors.New("ids list is empty")
	ErrDraftNotReady      = errors.New("draft is not ready to be sent")
	ErrDraftValidation    = errors.New("draft must contain at least one of: header, body, receivers")
	ErrDraftsLimit        = errors.New("drafts limit reached")
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrStorageUnavailable = errors.New("object storage is not configured")
)

// ErrSavedAsDraft возвращается когда письмо не удалось отправить через Postfix,
// но оно было автоматически сохранено как черновик.
type ErrSavedAsDraft struct {
	DraftID int64
}

func (e *ErrSavedAsDraft) Error() string {
	return fmt.Sprintf("sending failed, saved as draft: %d", e.DraftID)
}

type ErrRecipientNotFound struct {
	Email string
}

func (e *ErrRecipientNotFound) Error() string {
	return fmt.Sprintf("recipient not found: %s", e.Email)
}

func MapRepositoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrDuplicate):
		return ErrConflict
	case errors.Is(err, repository.ErrForeignKey):
		return ErrBadRequest
	case errors.Is(err, repository.ErrUserNotFound):
		return ErrUserNotFound
	case errors.Is(err, repository.ErrReceiverAdd):
		return ErrNoValidReceivers
	case errors.Is(err, repository.ErrMailNotFound):
		return ErrEmailNotFound
	case errors.Is(err, repository.ErrDraftNotFound):
		return ErrEmailNotFound
	case errors.Is(err, repository.ErrAccessDenied):
		return ErrAccessDenied
	default:
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return ErrUserNotFound
			case codes.PermissionDenied:
				return ErrAccessDenied
			case codes.InvalidArgument:
				return ErrBadRequest
			}
		}

		return err
	}
}
