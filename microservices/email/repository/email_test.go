package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --------------- helpers ---------------
func newTestRepo(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return &Repository{db: db}, mock
}

// --------------- GetUsersByEmails ---------------
func TestRepository_GetUsersByEmails(t *testing.T) {
	t.Run("success multiple", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		emails := []string{"a@smail.ru", "b@smail.ru"}

		rows := sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
			AddRow(1, "a@smail.ru", "Alice", "Smith").
			AddRow(2, "b@smail.ru", "Bob", "Brown")
		mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1, \$2\)`).
			WithArgs("a@smail.ru", "b@smail.ru").
			WillReturnRows(rows)

		users, err := repo.GetUsersByEmails(ctx, emails)
		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, "a@smail.ru", users[0].Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty input", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		users, err := repo.GetUsersByEmails(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, users)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))

		_, err := repo.GetUsersByEmails(ctx, []string{"x"})
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- SaveEmail ---------------
func TestRepository_SaveEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		email := models.Email{SenderID: 1, Header: "h", Body: "b"}
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(1), "h", "b").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

		id, err := repo.SaveEmail(ctx, email)
		require.NoError(t, err)
		assert.Equal(t, int64(10), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`INSERT`).WillReturnError(errors.New("fail"))

		_, err := repo.SaveEmail(ctx, models.Email{})
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- SaveEmailWithTx ---------------
func TestRepository_SaveEmailWithTx(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		email := models.Email{SenderID: 2, Header: "subj", Body: "text"}

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO emails`).
			WithArgs(int64(2), "subj", "text").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))
		mock.ExpectCommit()

		tx, err := repo.db.BeginTx(ctx, nil)
		require.NoError(t, err)
		id, err := repo.SaveEmailWithTx(ctx, tx, email)
		require.NoError(t, err)
		assert.Equal(t, int64(20), id)

		err = tx.Commit()
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT`).WillReturnError(errors.New("db fail"))
		mock.ExpectRollback()

		tx, _ := repo.db.BeginTx(ctx, nil)
		_, err := repo.SaveEmailWithTx(ctx, tx, models.Email{})
		assert.Error(t, err)
		tx.Rollback()
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- AddEmailUserWithTx ---------------
func TestRepository_AddEmailUserWithTx(t *testing.T) {
	t.Run("sender case", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_emails \(user_id, email_id, is_sender, is_spam\) VALUES \(\$1, \$2, true, false\)`).
			WithArgs(int64(5), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.AddEmailUserWithTx(ctx, tx, 100, 5, true)
		require.NoError(t, err)
		tx.Commit()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("receiver case", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO user_emails`).
			WithArgs(int64(5), int64(200)). // userID, emailID
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.AddEmailUserWithTx(ctx, tx, 200, 5, false)
		require.NoError(t, err)
		tx.Commit()
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		mock.ExpectBegin()
		mock.ExpectExec(`INSERT`).WillReturnError(errors.New("x"))
		mock.ExpectRollback()

		tx, _ := repo.db.BeginTx(ctx, nil)
		err := repo.AddEmailUserWithTx(ctx, tx, 100, 5, true)
		assert.Error(t, err)
		tx.Rollback()
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- CheckUserEmailExists ---------------
func TestRepository_CheckUserEmailExists(t *testing.T) {
	t.Run("exists true", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		exists, err := repo.CheckUserEmailExists(ctx, 10, 20)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exists false", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(10), int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		exists, err := repo.CheckUserEmailExists(ctx, 10, 20)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(errors.New("db error"))

		_, err := repo.CheckUserEmailExists(ctx, 10, 20)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- CheckEmailAccess ---------------
func TestRepository_CheckEmailAccess(t *testing.T) {
	t.Run("access granted", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1), int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		err := repo.CheckEmailAccess(ctx, 1, 2)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("access denied", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).
			WithArgs(int64(1), int64(2)).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		err := repo.CheckEmailAccess(ctx, 1, 2)
		assert.ErrorIs(t, err, ErrAccessDenied)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(errors.New("fail"))

		err := repo.CheckEmailAccess(ctx, 1, 2)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetEmailByID ---------------
func TestRepository_GetEmailByID(t *testing.T) {
	now := time.Now()
	t.Run("success with receivers", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path", "receivers_emails"}).
			AddRow(100, 5, "Subj", "Body", now, "/img.jpg", "{recv1@smail.ru,recv2@smail.ru}")
		mock.ExpectQuery(`SELECT (.+) FROM emails`).
			WithArgs(int64(100)).
			WillReturnRows(rows)

		email, err := repo.GetEmailByID(ctx, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(100), email.ID)
		assert.Equal(t, "/img.jpg", email.SenderImagePath)
		assert.Equal(t, []string{"recv1@smail.ru", "recv2@smail.ru"}, email.ReceiversEmails)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)

		_, err := repo.GetEmailByID(ctx, 99)
		assert.ErrorIs(t, err, ErrMailNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WithArgs(int64(1)).WillReturnError(errors.New("crash"))

		_, err := repo.GetEmailByID(ctx, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetEmailsByReceiver ---------------
func TestRepository_GetEmailsByReceiver(t *testing.T) {
	now := time.Now()
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "is_starred", "is_spam", "is_deleted", "received_at", "receivers_emails"}).
			AddRow(10, 3, "Hello", "World", now, false, true, false, false, now.Add(-time.Hour), "{a@smail.ru}")
		mock.ExpectQuery(`SELECT (.+) FROM user_emails ue JOIN emails e`).
			WithArgs(int64(42), 20, 0). // limit,offset after normPage
			WillReturnRows(rows)

		emails, err := repo.GetEmailsByReceiver(ctx, 42, 20, 0)
		require.NoError(t, err)
		require.Len(t, emails, 1)
		assert.Equal(t, int64(10), emails[0].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WithArgs(int64(42), 20, 0).WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "is_starred", "is_spam", "is_deleted", "received_at", "receivers_emails"}))
		list, err := repo.GetEmailsByReceiver(ctx, 42, 20, 0)
		require.NoError(t, err)
		assert.Empty(t, list)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db"))
		_, err := repo.GetEmailsByReceiver(ctx, 42, 20, 0)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetEmailsBySender ---------------
func TestRepository_GetEmailsBySender(t *testing.T) {
	now := time.Now()
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "is_starred", "is_spam", "is_deleted", "received_at", "receivers_emails"}).
			AddRow(11, 1, "Sent", "msg", now, false, false, false, false, now, "{to@smail.ru}")
		mock.ExpectQuery(`SELECT (.+) FROM user_emails ue JOIN emails e`).
			WithArgs(int64(2), 10, 5).
			WillReturnRows(rows)

		emails, err := repo.GetEmailsBySender(ctx, 2, 10, 5)
		require.NoError(t, err)
		require.Len(t, emails, 1)
		assert.Equal(t, int64(11), emails[0].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))
		_, err := repo.GetEmailsBySender(ctx, 1, 10, 0)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- Counts ---------------
func TestRepository_GetEmailsCount(t *testing.T) {
	repo, mock := newTestRepo(t)
	ctx := context.Background()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.GetEmailsCount(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 42, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUnreadEmailsCount(t *testing.T) {
	repo, mock := newTestRepo(t)
	ctx := context.Background()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := repo.GetUnreadEmailsCount(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetSenderEmailsCount(t *testing.T) {
	repo, mock := newTestRepo(t)
	ctx := context.Background()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))

	count, err := repo.GetSenderEmailsCount(ctx, 99)
	require.NoError(t, err)
	assert.Equal(t, 8, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --------------- MarkEmailAsRead / MarkEmailAsUnRead ---------------
func TestRepository_MarkEmailAsRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails SET is_read = \$3, updated_at = NOW\(\) WHERE email_id = \$1 AND user_id = \$2`).
			WithArgs(int64(1), int64(2), true).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.MarkEmailAsRead(ctx, 1, 2)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(99), int64(1), true).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.MarkEmailAsRead(ctx, 99, 1)
		assert.ErrorIs(t, err, ErrMailNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails`).WillReturnError(errors.New("fail"))
		err := repo.MarkEmailAsRead(ctx, 1, 1)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_MarkEmailAsUnRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails SET is_read = \$3, updated_at = NOW\(\)`).
			WithArgs(int64(10), int64(20), false).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.MarkEmailAsUnRead(ctx, 10, 20)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectExec(`UPDATE user_emails`).
			WithArgs(int64(10), int64(20), false).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.MarkEmailAsUnRead(ctx, 10, 20)
		assert.ErrorIs(t, err, ErrMailNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// --------------- GetEmailsByIDs ---------------
func TestRepository_GetEmailsByIDs(t *testing.T) {
	now := time.Now()
	t.Run("success", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		ids := []int64{10, 20}
		rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "is_starred", "is_spam", "is_deleted", "received_at", "receivers_emails"}).
			AddRow(10, 5, "H", "B", now, true, false, false, false, now, "{x@smail.ru}").
			AddRow(20, 6, "H2", "B2", now, false, true, false, false, now, "{}")

		mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN user_emails ue ON ue.email_id = e.id WHERE ue.user_id = \$1 AND e.id IN \(\$2,\$3\)`).
			WithArgs(int64(7), int64(10), int64(20)).
			WillReturnRows(rows)

		emails, err := repo.GetEmailsByIDs(ctx, ids, 7)
		require.NoError(t, err)
		require.Len(t, emails, 2)
		assert.Equal(t, int64(10), emails[0].ID)
		assert.Equal(t, int64(20), emails[1].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty IDs", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		list, err := repo.GetEmailsByIDs(ctx, []int64{}, 1)
		require.NoError(t, err)
		assert.Empty(t, list)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		ctx := context.Background()
		mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))
		_, err := repo.GetEmailsByIDs(ctx, []int64{1}, 2)
		assert.ErrorIs(t, err, ErrQueryFail)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
