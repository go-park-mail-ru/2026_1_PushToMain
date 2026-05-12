package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/models"
	"github.com/jackc/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	repo := New(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestRepository_CreateFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()
		folder := models.Folder{UserID: 1, Name: "inbox"}

		mock.ExpectQuery(`INSERT INTO folders`).
			WithArgs(int64(1), "inbox").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

		id, err := repo.CreateFolder(ctx, folder)
		require.NoError(t, err)
		assert.Equal(t, int64(42), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()
		folder := models.Folder{UserID: 1, Name: "inbox"}

		mock.ExpectQuery(`INSERT INTO folders`).
			WithArgs(int64(1), "inbox").
			WillReturnError(&pgconn.PgError{Code: "23505"})

		id, err := repo.CreateFolder(ctx, folder)
		assert.ErrorIs(t, err, ErrDuplicate)
		assert.Equal(t, int64(0), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()
		folder := models.Folder{UserID: 1, Name: "test"}

		mock.ExpectQuery(`INSERT INTO folders`).
			WithArgs(int64(1), "test").
			WillReturnError(errors.New("connection lost"))

		id, err := repo.CreateFolder(ctx, folder)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create folder for user 1")
		assert.Equal(t, int64(0), id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetFolderByID(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at"}).
			AddRow(1, 10, "inbox", now, now)
		mock.ExpectQuery(`SELECT id, user_id, name, created_at, updated_at FROM folders WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(rows)

		folder, err := repo.GetFolderByID(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), folder.ID)
		assert.Equal(t, int64(10), folder.UserID)
		assert.Equal(t, "inbox", folder.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT`).WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)

		folder, err := repo.GetFolderByID(ctx, 99)
		assert.ErrorIs(t, err, ErrFolderNotFound)
		assert.Nil(t, folder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT`).WithArgs(int64(1)).WillReturnError(errors.New("db down"))

		folder, err := repo.GetFolderByID(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get folder by id 1")
		assert.Nil(t, folder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_UpdateFolderName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`UPDATE folders SET name = \$1, updated_at = NOW\(\) WHERE id = \$2`).
			WithArgs("new_name", int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.UpdateFolderName(ctx, 1, "new_name")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`UPDATE folders`).
			WithArgs("new_name", int64(99)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.UpdateFolderName(ctx, 99, "new_name")
		assert.ErrorIs(t, err, ErrFolderNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`UPDATE folders`).
			WithArgs("existing", int64(1)).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err = repo.UpdateFolderName(ctx, 1, "existing")
		assert.ErrorIs(t, err, ErrDuplicate)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`UPDATE folders`).
			WithArgs("name", int64(1)).
			WillReturnError(errors.New("db down"))

		err = repo.UpdateFolderName(ctx, 1, "name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update folder name for folder 1")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RowsAffected error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`UPDATE folders`).
			WithArgs("name", int64(1)).
			WillReturnResult(sqlmock.NewErrorResult(errors.New("driver error")))

		err = repo.UpdateFolderName(ctx, 1, "name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected for folder 1")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetFolderByName(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id", "user_id", "name", "created_at", "updated_at"}).
			AddRow(5, 10, "inbox", now, now)
		mock.ExpectQuery(`SELECT id, user_id, name, created_at, updated_at FROM folders WHERE user_id = \$1 AND name = \$2`).
			WithArgs(int64(10), "inbox").
			WillReturnRows(rows)

		folder, err := repo.GetFolderByName(ctx, 10, "inbox")
		require.NoError(t, err)
		assert.Equal(t, int64(5), folder.ID)
		assert.Equal(t, int64(10), folder.UserID)
		assert.Equal(t, "inbox", folder.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT`).WithArgs(int64(10), "missing").WillReturnError(sql.ErrNoRows)

		folder, err := repo.GetFolderByName(ctx, 10, "missing")
		assert.ErrorIs(t, err, ErrFolderNotFound)
		assert.Nil(t, folder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT`).WithArgs(int64(10), "inbox").WillReturnError(errors.New("db down"))

		folder, err := repo.GetFolderByName(ctx, 10, "inbox")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get folder by name 'inbox' for user 10")
		assert.Nil(t, folder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_CountUserFolders(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folders WHERE user_id = \$1`).
			WithArgs(int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.CountUserFolders(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT`).WithArgs(int64(10)).WillReturnError(errors.New("db down"))

		count, err := repo.CountUserFolders(ctx, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count folders for user 10")
		assert.Equal(t, 0, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_CountEmailsInFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_emails WHERE folder_id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		count, err := repo.CountEmailsInFolder(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 10, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT COUNT`).WithArgs(int64(1)).WillReturnError(errors.New("db down"))

		count, err := repo.CountEmailsInFolder(ctx, 1)
		assert.Error(t, err)
		assert.Equal(t, 0, count)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_AddEmailToFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`INSERT INTO folder_emails`).
			WithArgs(int64(1), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.AddEmailToFolder(ctx, 1, 100)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`INSERT INTO folder_emails`).
			WithArgs(int64(1), int64(100)).
			WillReturnError(errors.New("db down"))

		err = repo.AddEmailToFolder(ctx, 1, 100)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_DeleteEmailFromFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folder_emails WHERE folder_id = \$1 AND email_id = \$2`).
			WithArgs(int64(1), int64(100)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.DeleteEmailFromFolder(ctx, 1, 100)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows affected", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folder_emails`).
			WithArgs(int64(1), int64(999)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.DeleteEmailFromFolder(ctx, 1, 999)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folder_emails`).
			WithArgs(int64(1), int64(100)).
			WillReturnError(errors.New("db down"))

		err = repo.DeleteEmailFromFolder(ctx, 1, 100)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_DeleteFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folders WHERE id = \$1 AND user_id = \$2`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.DeleteFolder(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folders`).
			WithArgs(int64(99), int64(10)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = repo.DeleteFolder(ctx, 99, 10)
		assert.ErrorIs(t, err, ErrFolderNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folders`).
			WithArgs(int64(1), int64(10)).
			WillReturnError(errors.New("db down"))

		err = repo.DeleteFolder(ctx, 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete folder")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RowsAffected error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectExec(`DELETE FROM folders`).
			WithArgs(int64(1), int64(10)).
			WillReturnResult(sqlmock.NewErrorResult(errors.New("driver error")))

		err = repo.DeleteFolder(ctx, 1, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepository_GetFolderEmailIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"email_id"}).
			AddRow(int64(100)).
			AddRow(int64(200)).
			AddRow(int64(300))
		mock.ExpectQuery(`SELECT email_id FROM folder_emails WHERE folder_id = \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
			WithArgs(int64(1), 10, 0).
			WillReturnRows(rows)

		ids, err := repo.GetFolderEmailIDs(ctx, 1, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, []int64{100, 200, 300}, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"email_id"})
		mock.ExpectQuery(`SELECT email_id FROM folder_emails`).
			WithArgs(int64(1), 10, 0).
			WillReturnRows(rows)

		ids, err := repo.GetFolderEmailIDs(ctx, 1, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		mock.ExpectQuery(`SELECT email_id FROM folder_emails`).
			WithArgs(int64(1), 10, 0).
			WillReturnError(errors.New("db down"))

		ids, err := repo.GetFolderEmailIDs(ctx, 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := New(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"email_id"}).
			AddRow("not_an_int")
		mock.ExpectQuery(`SELECT email_id FROM folder_emails`).
			WithArgs(int64(1), 10, 0).
			WillReturnRows(rows)

		ids, err := repo.GetFolderEmailIDs(ctx, 1, 10, 0)
		assert.Error(t, err)
		assert.Nil(t, ids)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
