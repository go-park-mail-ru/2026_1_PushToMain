package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_SetStarredBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails SET is_starred = \$2, updated_at = NOW\(\) WHERE user_id = \$1 AND email_id IN \(\$3,\$4\)`).
			WithArgs(int64(1), true, int64(10), int64(20)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		err := repo.SetStarredBatch(ctx, 1, []int64{10, 20}, true)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.SetStarredBatch(ctx, 1, []int64{}, true)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails`).WillReturnError(errors.New("fail"))
		err := repo.SetStarredBatch(ctx, 1, []int64{1}, true)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_SetTrashedBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails SET is_deleted = \$2, updated_at = NOW\(\) WHERE user_id = \$1 AND email_id IN \(\$3,\$4\)`).
			WithArgs(int64(99), true, int64(7), int64(8)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		err := repo.SetTrashedBatch(ctx, 99, []int64{7, 8}, true)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.SetTrashedBatch(ctx, 1, []int64{}, false)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_SetSpamBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails SET is_spam = \$2, updated_at = NOW\(\) WHERE user_id = \$1 AND is_sender = false AND email_id IN \(\$3,\$4\)`).
			WithArgs(int64(5), true, int64(1), int64(2)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		err := repo.SetSpamBatch(ctx, 5, []int64{1, 2}, true)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.SetSpamBatch(ctx, 1, nil, false)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_MarkSendersAsSpamBatch(t *testing.T) {
	t.Run("success full flow", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()

		// senderIDsForReceivedEmails
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT DISTINCT e.sender_id
        FROM emails e
        JOIN user_emails ue ON ue.email_id = e.id
        WHERE ue.user_id = $1 AND ue.is_sender = false
          AND e.id IN ($2,$3)`)).
			WithArgs(int64(1), int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"sender_id"}).AddRow(100).AddRow(200))

		// insertSpamSenders
		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO spam_senders (user_id, sender_id)
        VALUES ($1, $2),($1, $3)
        ON CONFLICT (user_id, sender_id) DO NOTHING`)).
			WithArgs(int64(1), int64(100), int64(200)).
			WillReturnResult(sqlmock.NewResult(0, 2))

		// markEmailsFromSendersAsSpam
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE user_emails
        SET is_spam = true, updated_at = NOW()
        WHERE user_id = $1 AND is_sender = false AND is_spam = false
          AND email_id IN (
              SELECT id FROM emails WHERE sender_id IN ($2,$3)
          )`)).
			WithArgs(int64(1), int64(100), int64(200)).
			WillReturnResult(sqlmock.NewResult(0, 3))

		mock.ExpectCommit()

		err := repo.MarkSendersAsSpamBatch(ctx, 1, []int64{10, 20})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty email list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.MarkSendersAsSpamBatch(ctx, 1, []int64{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no senders found -> commits and returns", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT DISTINCT e\.sender_id`).
			WithArgs(int64(1), int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"sender_id"}))
		mock.ExpectCommit()

		err := repo.MarkSendersAsSpamBatch(ctx, 1, []int64{10})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin tx fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		err := repo.MarkSendersAsSpamBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query fails inside tx", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("query fail"))
		mock.ExpectRollback()

		err := repo.MarkSendersAsSpamBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_HardDeleteBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`DELETE FROM user_emails WHERE user_id = \$1 AND email_id IN \(\$2,\$3\)`).
			WithArgs(int64(77), int64(33), int64(44)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		err := repo.HardDeleteBatch(ctx, 77, []int64{33, 44})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.HardDeleteBatch(ctx, 1, []int64{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`DELETE`).WillReturnError(errors.New("fail"))
		err := repo.HardDeleteBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_UnmarkSendersAsSpamBatch(t *testing.T) {
	t.Run("success full flow", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		// senderIDsForEmails
		mock.ExpectQuery(`SELECT DISTINCT sender_id FROM emails WHERE id IN \(\$1,\$2\)`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"sender_id"}).AddRow(300))
		// deleteSpamSenders
		mock.ExpectExec(`DELETE FROM spam_senders WHERE user_id = \$1 AND sender_id IN \(\$2\)`).
			WithArgs(int64(42), int64(300)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// unmarkEmailsFromSendersAsSpam
		mock.ExpectExec(`UPDATE user_emails SET is_spam = false, updated_at = NOW\(\) WHERE user_id = \$1 AND is_sender = false AND is_spam = true AND email_id IN \(SELECT id FROM emails WHERE sender_id IN \(\$2\)\)`).
			WithArgs(int64(42), int64(300)).
			WillReturnResult(sqlmock.NewResult(0, 5))
		mock.ExpectCommit()

		err := repo.UnmarkSendersAsSpamBatch(ctx, 42, []int64{10, 20})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty email list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.UnmarkSendersAsSpamBatch(ctx, 1, []int64{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no senders found -> commit", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT DISTINCT sender_id FROM emails`).
			WithArgs(int64(99)).
			WillReturnRows(sqlmock.NewRows([]string{"sender_id"}))
		mock.ExpectCommit()
		err := repo.UnmarkSendersAsSpamBatch(ctx, 1, []int64{99})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin tx fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin().WillReturnError(errors.New("tx err"))
		err := repo.UnmarkSendersAsSpamBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query fails inside tx", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("oops"))
		mock.ExpectRollback()
		err := repo.UnmarkSendersAsSpamBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetUserEmailFlags(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "email_id", "user_id", "is_sender", "is_read", "is_deleted", "is_starred", "is_spam", "is_draft"}).
			AddRow(1, 100, 42, false, true, false, true, false, false)
		mock.ExpectQuery(`SELECT id, email_id, user_id, is_sender, is_read, is_deleted, is_starred, is_spam, is_draft FROM user_emails WHERE email_id = \$1 AND user_id = \$2 AND is_sender = \$3`).
			WithArgs(int64(100), int64(42), false).
			WillReturnRows(rows)

		ue, err := repo.GetUserEmailFlags(ctx, 100, 42, false)
		require.NoError(t, err)
		assert.Equal(t, int64(1), ue.ID)
		assert.True(t, ue.IsRead)
		assert.True(t, ue.IsStarred)
		assert.False(t, ue.IsSpam)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WithArgs(int64(1), int64(1), true).WillReturnError(sql.ErrNoRows)
		_, err := repo.GetUserEmailFlags(ctx, 1, 1, true)
		assert.ErrorIs(t, err, ErrMailNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
		_, err := repo.GetUserEmailFlags(ctx, 1, 1, false)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
