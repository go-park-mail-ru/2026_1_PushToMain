package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --------------- CountDraftsByUser ---------------

func TestRepository_CountDraftsByUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountDraftsByUser(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT`).WillReturnError(errors.New("db error"))
		_, err := repo.CountDraftsByUser(ctx, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- CreateDraft ---------------

func TestRepository_CreateDraft(t *testing.T) {
	draft := models.Draft{
		SenderID:  10,
		Header:    "Subject",
		Body:      "Body",
		Receivers: []string{"a@smail.ru", "b@smail.ru"},
	}

	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO emails (sender_id, header, body) VALUES ($1, $2, $3) RETURNING id`)).
			WithArgs(int64(10), "Subject", "Body").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO user_emails (user_id, email_id, is_sender, is_draft) VALUES ($1, $2, true, true)`)).
			WithArgs(int64(10), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(
			`INSERT INTO draft_receivers (email_id, receiver_email) VALUES ($1, $2),($1, $3)`)).
			WithArgs(int64(100), "a@smail.ru", "b@smail.ru").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		id, err := repo.CreateDraft(ctx, draft)
		require.NoError(t, err)
		assert.Equal(t, int64(100), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with no receivers", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		draftNoRecv := models.Draft{SenderID: 10, Header: "H", Body: "B", Receivers: nil}

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(10), "H", "B").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(200))
		mock.ExpectExec(`INSERT INTO user_emails`).
			WithArgs(int64(10), int64(200)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		id, err := repo.CreateDraft(ctx, draftNoRecv)
		require.NoError(t, err)
		assert.Equal(t, int64(200), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin tx fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		_, err := repo.CreateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("email insert fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		_, err := repo.CreateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrSaveEmail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user_emails insert fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(10), "Subject", "Body").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(`INSERT INTO user_emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		_, err := repo.CreateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrSaveEmail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("draft_receivers insert fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(10), "Subject", "Body").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(`INSERT INTO user_emails`).
			WithArgs(int64(10), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO draft_receivers`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		_, err := repo.CreateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrReceiverAdd)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(10), "Subject", "Body").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))
		mock.ExpectExec(`INSERT INTO user_emails`).
			WithArgs(int64(10), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO draft_receivers`).
			WithArgs(int64(100), "a@smail.ru", "b@smail.ru").
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit().WillReturnError(errors.New("commit fail"))

		_, err := repo.CreateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- UpdateDraft ---------------

func TestRepository_UpdateDraft(t *testing.T) {
	draft := models.Draft{
		ID:        100,
		SenderID:  10,
		Header:    "New Subj",
		Body:      "New Body",
		Receivers: []string{"c@smail.ru"},
	}

	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE emails e SET header = $1, body = $2 FROM user_emails ue WHERE e.id = $3 AND ue.email_id = e.id AND ue.user_id = $4 AND ue.is_sender = true AND ue.is_draft = true`)).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE user_emails SET updated_at = NOW\(\)`).
			WithArgs(int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers WHERE email_id = \$1`).
			WithArgs(int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO draft_receivers`).
			WithArgs(int64(100), "c@smail.ru").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateDraft(ctx, draft)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected (draft not found)", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrDraftNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error during update", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update user_emails fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE user_emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete receivers fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("insert receivers fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers`).
			WithArgs(int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO draft_receivers`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrReceiverAdd)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE emails`).
			WithArgs("New Subj", "New Body", int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers`).
			WithArgs(int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO draft_receivers`).
			WithArgs(int64(100), "c@smail.ru").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit err"))

		err := repo.UpdateDraft(ctx, draft)
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetDraftByID ---------------

func TestRepository_GetDraftByID(t *testing.T) {
	now := time.Now()
	updated := now.Add(-time.Hour)

	t.Run("success with receivers", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "updated_at", "receivers"}).
			AddRow(100, 10, "Subj", "Body", now, updated, "{x@smail.ru,y@smail.ru}")
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at, COALESCE(( SELECT array_agg(dr.receiver_email ORDER BY dr.id) FROM draft_receivers dr WHERE dr.email_id = e.id ), '{}'::text[]) AS receivers FROM emails e JOIN user_emails ue ON ue.email_id = e.id WHERE e.id = $1 AND ue.user_id = $2 AND ue.is_sender = true AND ue.is_draft = true`)).
			WithArgs(int64(100), int64(10)).
			WillReturnRows(rows)

		d, err := repo.GetDraftByID(ctx, 100, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(100), d.ID)
		assert.Equal(t, []string{"x@smail.ru", "y@smail.ru"}, d.Receivers)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WithArgs(int64(99), int64(1)).WillReturnError(sql.ErrNoRows)

		_, err := repo.GetDraftByID(ctx, 99, 1)
		assert.ErrorIs(t, err, ErrDraftNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))

		_, err := repo.GetDraftByID(ctx, 1, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetDrafts ---------------

func TestRepository_GetDrafts(t *testing.T) {
	now := time.Now()
	updated := now.Add(-time.Hour)

	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "updated_at", "receivers"}).
			AddRow(1, 10, "H1", "B1", now, updated, "{r1@smail.ru}").
			AddRow(2, 10, "H2", "B2", now, updated, "{}")
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT e.id, e.sender_id, e.header, e.body, e.created_at, ue.updated_at, COALESCE(( SELECT array_agg(dr.receiver_email ORDER BY dr.id) FROM draft_receivers dr WHERE dr.email_id = e.id ), '{}'::text[]) AS receivers FROM emails e JOIN user_emails ue ON ue.email_id = e.id WHERE ue.user_id = $1 AND ue.is_sender = true AND ue.is_draft = true ORDER BY ue.updated_at DESC LIMIT $2 OFFSET $3`)).
			WithArgs(int64(10), 20, 0). // limit normalized from 20 to 20, offset 0
			WillReturnRows(rows)

		drafts, err := repo.GetDrafts(ctx, 10, 20, 0)
		require.NoError(t, err)
		require.Len(t, drafts, 2)
		assert.Equal(t, int64(1), drafts[0].ID)
		assert.Equal(t, []string{"r1@smail.ru"}, drafts[0].Receivers)
		assert.Empty(t, drafts[1].Receivers)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "updated_at", "receivers"})
		mock.ExpectQuery(`SELECT`).WithArgs(int64(1), 20, 0).WillReturnRows(rows)

		drafts, err := repo.GetDrafts(ctx, 1, 20, 0)
		require.NoError(t, err)
		assert.Empty(t, drafts)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))

		_, err := repo.GetDrafts(ctx, 1, 20, 0)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "updated_at", "receivers"}).
			AddRow("bad", 10, "H", "B", now, updated, "{}") // id is string, will fail Scan
		mock.ExpectQuery(`SELECT`).WithArgs(int64(10), 20, 0).WillReturnRows(rows)

		_, err := repo.GetDrafts(ctx, 10, 20, 0)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows iteration error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "updated_at", "receivers"}).
			AddRow(1, 10, "H", "B", now, updated, "{}").
			RowError(0, errors.New("row fail"))
		mock.ExpectQuery(`SELECT`).WithArgs(int64(10), 20, 0).WillReturnRows(rows)

		_, err := repo.GetDrafts(ctx, 10, 20, 0)
		assert.ErrorIs(t, err, ErrQueryFail) // rows.Err() returns ErrQueryFail? Actually it returns the row error directly, but the code returns ErrQueryFail. We'll check that error is not nil.
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- DeleteDraftsBatch ---------------

func TestRepository_DeleteDraftsBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM user_emails
        WHERE user_id = $1 AND is_sender = true AND is_draft = true
          AND email_id IN ($2,$3)`)).
			WithArgs(int64(1), int64(10), int64(20)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM emails WHERE id IN ($2,$3)`)).
			WithArgs(int64(10), int64(20)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.DeleteDraftsBatch(ctx, 1, []int64{10, 20})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty IDs", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		err := repo.DeleteDraftsBatch(ctx, 1, []int64{})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin tx fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin().WillReturnError(errors.New("tx err"))
		err := repo.DeleteDraftsBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete user_emails fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.DeleteDraftsBatch(ctx, 1, []int64{1})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete emails fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_emails`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM emails`).WillReturnError(errors.New("fail"))
		mock.ExpectRollback()

		err := repo.DeleteDraftsBatch(ctx, 1, []int64{10})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM user_emails`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM emails`).
			WithArgs(int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit err"))

		err := repo.DeleteDraftsBatch(ctx, 1, []int64{10})
		assert.ErrorIs(t, err, ErrTransactionFailed)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- MarkDraftAsSentTx ---------------

func TestRepository_MarkDraftAsSentTx(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE user_emails SET is_draft = false, updated_at = NOW\(\) WHERE email_id = \$1 AND user_id = \$2 AND is_sender = true AND is_draft = true`).
			WithArgs(int64(100), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers WHERE email_id = \$1`).
			WithArgs(int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.MarkDraftAsSentTx(ctx, tx, 100, 10)
		require.NoError(t, err)
		tx.Commit()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(1), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.MarkDraftAsSentTx(ctx, tx, 1, 1)
		assert.ErrorIs(t, err, ErrDraftNotFound)
		tx.Rollback()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE user_emails`).WillReturnError(errors.New("fail"))
		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.MarkDraftAsSentTx(ctx, tx, 1, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		tx.Rollback()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete draft_receivers fails", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(1), int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM draft_receivers`).WillReturnError(errors.New("fail"))
		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.MarkDraftAsSentTx(ctx, tx, 1, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		tx.Rollback()
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
