package service

import (
	"errors"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailNotFound    = errors.New("email not found")
	ErrNoValidReceivers = errors.New("no valid receivers found")
	ErrAccessDenied     = errors.New("don't have access to this email")
	ErrTransaction      = errors.New("transaction failed")
	ErrConflict         = errors.New("conflict")
	ErrBadRequest       = errors.New("bad request")
	ErrEmptyIDs         = errors.New("ids list is empty")
	ErrDraftNotReady    = errors.New("draft is not ready to be sent")
	ErrDraftValidation  = errors.New("draft must contain at least one of: header, body, receivers")
	ErrDraftsLimit      = errors.New("drafts limit reached")
)

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
		return err
	}
}
