package repository

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	UniqueViolation     = "23505"
	ForeignKeyViolation = "23503"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrQueryFail         = errors.New("query failed")
	ErrMailNotFound      = errors.New("email not found")
	ErrTransactionFailed = errors.New("transaction failed")
	ErrSaveEmail         = errors.New("failed to save email")
	ErrReceiverAdd       = errors.New("failed to add receivers")
	ErrDuplicate         = errors.New("record already exists")
	ErrForeignKey        = errors.New("related record not found")
	ErrAccessDenied      = errors.New("have no access")
	ErrDraftNotFound     = errors.New("draft not found")
	ErrEmailInTrash      = errors.New("email is in trash")
	ErrCannotSpamSelf    = errors.New("cannot spam self-sent email")
)

func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case UniqueViolation:
			return ErrDuplicate
		case ForeignKeyViolation:
			return ErrForeignKey
		}
	}
	return ErrSaveEmail
}

func mapPgErrorReceiver(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case UniqueViolation:
			return ErrDuplicate
		case ForeignKeyViolation:
			return ErrForeignKey
		}
	}
	return ErrReceiverAdd
}

func parsePgTextArray(s string) []string {
	s = strings.Trim(s, "{}")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func normPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
