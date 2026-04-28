package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	repo := New(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.userDb)
}

func TestRepository_UpdateProfile(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		nameParam   string
		surname     string
		mockSetup   func(mock sqlmock.Sqlmock)
		expectedErr error
	}{
		{
			name:      "success",
			userID:    1,
			nameParam: "John",
			surname:   "Doe",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET name = \$1, surname = \$2, updated_at = NOW\(\) WHERE id = \$3`).
					WithArgs("John", "Doe", int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:      "no rows affected - user not found",
			userID:    99,
			nameParam: "John",
			surname:   "Doe",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("John", "Doe", int64(99)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: ErrUserNotFound,
		},
		{
			name:      "query error",
			userID:    1,
			nameParam: "John",
			surname:   "Doe",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("John", "Doe", int64(1)).
					WillReturnError(errors.New("connection lost"))
			},
			expectedErr: ErrQueryError,
		},
		{
			name:      "RowsAffected error",
			userID:    1,
			nameParam: "John",
			surname:   "Doe",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("John", "Doe", int64(1)).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("driver error")))
			},
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			err = repo.UpdateProfile(context.Background(), tt.userID, tt.nameParam, tt.surname, nil, nil)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_UpdateAvatar(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		imagePath   string
		mockSetup   func(mock sqlmock.Sqlmock)
		expectedErr error
	}{
		{
			name:      "success",
			userID:    1,
			imagePath: "/avatars/1.jpg",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET image_path = \$1 WHERE id = \$2`).
					WithArgs("/avatars/1.jpg", int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:      "no rows affected",
			userID:    99,
			imagePath: "/avatars/99.jpg",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("/avatars/99.jpg", int64(99)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: ErrUserNotFound,
		},
		{
			name:      "query error",
			userID:    1,
			imagePath: "/avatars/1.jpg",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(errors.New("db error"))
			},
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			err = repo.UpdateAvatar(context.Background(), tt.userID, tt.imagePath)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("nil database", func(t *testing.T) {
		repo := &Repository{userDb: nil}
		err := repo.UpdateAvatar(context.Background(), 1, "path")
		assert.ErrorIs(t, err, ErrUserDbNotInited)
	})
}

func TestRepository_Save(t *testing.T) {
	tests := []struct {
		name        string
		user        models.User
		mockSetup   func(mock sqlmock.Sqlmock)
		expectedID  int64
		expectedErr error
	}{
		{
			name: "success",
			user: models.User{
				Email:     "test@example.com",
				Password:  "hash",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/default.jpg",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO users`).
					WithArgs("test@example.com", "hash", "John", "Doe", "/default.jpg").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
			},
			expectedID:  42,
			expectedErr: nil,
		},
		{
			name: "query error",
			user: models.User{
				Email: "test@example.com",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT`).
					WillReturnError(errors.New("duplicate key"))
			},
			expectedID:  0,
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			id, err := repo.Save(context.Background(), tt.user)
			assert.Equal(t, tt.expectedID, id)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("nil database", func(t *testing.T) {
		repo := &Repository{userDb: nil}
		_, err := repo.Save(context.Background(), models.User{})
		assert.ErrorIs(t, err, ErrUserDbNotInited)
	})
}

func TestRepository_FindByEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		mockSetup   func(mock sqlmock.Sqlmock)
		expected    *models.User
		expectedErr error
	}{
		{
			name:  "success",
			email: "john@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password_hash", "name", "surname", "image_path"}).
					AddRow(1, "hash", "John", "Doe", "/avatar.jpg")
				mock.ExpectQuery(`SELECT id, password_hash, name, surname, image_path FROM users WHERE email = \$1`).
					WithArgs("john@example.com").
					WillReturnRows(rows)
			},
			expected: &models.User{
				ID:        1,
				Email:     "john@example.com",
				Password:  "hash",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatar.jpg",
			},
			expectedErr: nil,
		},
		{
			name:  "user not found",
			email: "missing@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).
					WithArgs("missing@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: ErrUserNotFound,
		},
		{
			name:  "query error",
			email: "error@example.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).
					WillReturnError(errors.New("connection lost"))
			},
			expected:    nil,
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			user, err := repo.FindByEmail(context.Background(), tt.email)
			assert.Equal(t, tt.expected, user)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("nil database", func(t *testing.T) {
		repo := &Repository{userDb: nil}
		_, err := repo.FindByEmail(context.Background(), "test@example.com")
		assert.ErrorIs(t, err, ErrUserDbNotInited)
	})
}

func TestRepository_FindByID(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		mockSetup   func(mock sqlmock.Sqlmock)
		expected    *models.User
		expectedErr error
	}{
		{
			name:   "success",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "surname", "image_path"}).
					AddRow(1, "john@example.com", "hash", "John", "Doe", "/avatar.jpg")
				mock.ExpectQuery(`SELECT id, email, password_hash, name, surname, image_path FROM users WHERE id = \$1`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			expected: &models.User{
				ID:        1,
				Email:     "john@example.com",
				Password:  "hash",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatar.jpg",
			},
			expectedErr: nil,
		},
		{
			name:   "user not found",
			userID: 99,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).
					WithArgs(int64(99)).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: ErrUserNotFound,
		},
		{
			name:   "query error",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).
					WillReturnError(errors.New("db down"))
			},
			expected:    nil,
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			user, err := repo.FindByID(context.Background(), tt.userID)
			assert.Equal(t, tt.expected, user)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("nil database", func(t *testing.T) {
		repo := &Repository{userDb: nil}
		_, err := repo.FindByID(context.Background(), 1)
		assert.ErrorIs(t, err, ErrUserDbNotInited)
	})
}

func TestRepository_UpdatePassword(t *testing.T) {
	tests := []struct {
		name         string
		userID       int64
		passwordHash string
		mockSetup    func(mock sqlmock.Sqlmock)
		expectedErr  error
	}{
		{
			name:         "success",
			userID:       1,
			passwordHash: "newhash",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users SET password_hash = \$1 WHERE id = \$2`).
					WithArgs("newhash", int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:         "no rows affected",
			userID:       99,
			passwordHash: "newhash",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WithArgs("newhash", int64(99)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: ErrUserNotFound,
		},
		{
			name:         "query error",
			userID:       1,
			passwordHash: "newhash",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnError(errors.New("connection lost"))
			},
			expectedErr: ErrQueryError,
		},
		{
			name:         "RowsAffected error",
			userID:       1,
			passwordHash: "newhash",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE users`).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("driver error")))
			},
			expectedErr: ErrQueryError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := &Repository{userDb: db}
			tt.mockSetup(mock)

			err = repo.UpdatePassword(context.Background(), tt.userID, tt.passwordHash)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("nil database", func(t *testing.T) {
		repo := &Repository{userDb: nil}
		err := repo.UpdatePassword(context.Background(), 1, "hash")
		assert.ErrorIs(t, err, ErrUserDbNotInited)
	})
}
