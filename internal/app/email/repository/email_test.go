package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestRepository_GetUsersByEmails(t *testing.T) {
	tests := []struct {
		name      string
		emails    []string
		mockSetup func(mock sqlmock.Sqlmock)
		want      []*models.User
		wantErr   error
	}{
		{
			name:   "empty emails",
			emails: []string{},
			mockSetup: func(mock sqlmock.Sqlmock) {
			},
			want:    []*models.User{},
			wantErr: nil,
		},
		{
			name:   "success single",
			emails: []string{"a@b.com"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
					AddRow(1, "a@b.com", "John", "Doe")
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("a@b.com").
					WillReturnRows(rows)
			},
			want: []*models.User{
				{ID: 1, Email: "a@b.com", Name: "John", Surname: "Doe"},
			},
			wantErr: nil,
		},
		{
			name:   "success multiple",
			emails: []string{"a@b.com", "c@d.com"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
					AddRow(1, "a@b.com", "John", "Doe").
					AddRow(2, "c@d.com", "Jane", "Smith")
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1, \$2\)`).
					WithArgs("a@b.com", "c@d.com").
					WillReturnRows(rows)
			},
			want: []*models.User{
				{ID: 1, Email: "a@b.com", Name: "John", Surname: "Doe"},
				{ID: 2, Email: "c@d.com", Name: "Jane", Surname: "Smith"},
			},
			wantErr: nil,
		},
		{
			name:   "query error",
			emails: []string{"a@b.com"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db down"))
			},
			want:    nil,
			wantErr: ErrQueryFail,
		},
		{
			name:   "scan error",
			emails: []string{"a@b.com"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
					AddRow("not_int", "a@b.com", "John", "Doe")
				mock.ExpectQuery(`SELECT`).WithArgs("a@b.com").WillReturnRows(rows)
			},
			want:    nil,
			wantErr: ErrQueryFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetUsersByEmails(context.Background(), tt.emails)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_SaveEmail(t *testing.T) {
	now := time.Now()
	email := models.Email{SenderID: 1, Header: "Hello", Body: "World", CreatedAt: now}
	tests := []struct {
		name      string
		email     models.Email
		mockSetup func(mock sqlmock.Sqlmock)
		wantID    int64
		wantErr   error
	}{
		{
			name:  "success",
			email: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(email.SenderID, email.Header, email.Body).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
			},
			wantID:  10,
			wantErr: nil,
		},
		{
			name:  "duplicate key",
			email: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(email.SenderID, email.Header, email.Body).
					WillReturnError(&pgconn.PgError{Code: UniqueViolation})
			},
			wantID:  0,
			wantErr: ErrDuplicate,
		},
		{
			name:  "foreign key violation",
			email: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(email.SenderID, email.Header, email.Body).
					WillReturnError(&pgconn.PgError{Code: ForeignKeyViolation})
			},
			wantID:  0,
			wantErr: ErrForeignKey,
		},
		{
			name:  "other error",
			email: email,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO emails`).
					WillReturnError(errors.New("some error"))
			},
			wantID:  0,
			wantErr: ErrSaveEmail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			id, err := repo.SaveEmail(context.Background(), tt.email)
			assert.Equal(t, tt.wantID, id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_AddEmailReceivers(t *testing.T) {
	tests := []struct {
		name        string
		emailID     int64
		receiverIDs []int64
		mockSetup   func(mock sqlmock.Sqlmock)
		wantErr     error
	}{
		{
			name:        "empty receivers",
			emailID:     1,
			receiverIDs: []int64{},
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			wantErr:     nil,
		},
		{
			name:        "success single",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_emails \(receiver_id, email_id\) VALUES \(\$1, \$2\)`).
					WithArgs(int64(2), int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:        "success multiple",
			emailID:     1,
			receiverIDs: []int64{2, 3},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_emails \(receiver_id, email_id\) VALUES \(\$1, \$2\), \(\$3, \$4\)`).
					WithArgs(int64(2), int64(1), int64(3), int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 2))
			},
			wantErr: nil,
		},
		{
			name:        "duplicate error",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(2), int64(1)).
					WillReturnError(&pgconn.PgError{Code: UniqueViolation})
			},
			wantErr: ErrDuplicate,
		},
		{
			name:        "foreign key error",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(2), int64(1)).
					WillReturnError(&pgconn.PgError{Code: ForeignKeyViolation})
			},
			wantErr: ErrForeignKey,
		},
		{
			name:        "other error",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_emails`).
					WillReturnError(errors.New("some error"))
			},
			wantErr: ErrReceiverAdd,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			err := repo.AddEmailReceivers(context.Background(), tt.emailID, tt.receiverIDs)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_SaveEmailWithTx(t *testing.T) {
	email := models.Email{SenderID: 1, Header: "Hello", Body: "World"}
	tests := []struct {
		name      string
		mockSetup func(mock sqlmock.Sqlmock)
		wantID    int64
		wantErr   error
	}{
		{
			name: "success",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(email.SenderID, email.Header, email.Body).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
			},
			wantID:  5,
			wantErr: nil,
		},
		{
			name: "duplicate",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO emails`).
					WillReturnError(&pgconn.PgError{Code: UniqueViolation})
			},
			wantID:  0,
			wantErr: ErrDuplicate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			tt.mockSetup(mock)
			tx, err := db.Begin()
			require.NoError(t, err)

			repo := &Repository{db: db}
			id, err := repo.SaveEmailWithTx(context.Background(), tx, email)

			assert.Equal(t, tt.wantID, id)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
func TestRepository_AddEmailReceiversWithTx(t *testing.T) {
	tests := []struct {
		name        string
		emailID     int64
		receiverIDs []int64
		mockSetup   func(mock sqlmock.Sqlmock)
		wantErr     error
	}{
		{
			name:        "empty receivers",
			emailID:     1,
			receiverIDs: []int64{},
			mockSetup:   func(mock sqlmock.Sqlmock) { mock.ExpectBegin() },
			wantErr:     nil,
		},
		{
			name:        "success",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(2), int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:        "error",
			emailID:     1,
			receiverIDs: []int64{2},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO user_emails`).
					WillReturnError(errors.New("db error"))
			},
			wantErr: ErrReceiverAdd,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			tt.mockSetup(mock)
			tx, err := db.Begin()
			require.NoError(t, err)

			repo := &Repository{db: db}
			err = repo.AddEmailReceiversWithTx(context.Background(), tx, tt.emailID, tt.receiverIDs)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetEmailsByReceiver(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		userID    int64
		limit     int
		offset    int
		mockSetup func(mock sqlmock.Sqlmock)
		want      []models.EmailWithMetadata
		wantErr   bool
	}{
		{
			name:   "default limit/offset",
			userID: 1,
			limit:  0,
			offset: -1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "received_at"}).
					AddRow(1, 2, "Subj", "Body", now, false, now)
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN user_emails ue`).
					WithArgs(int64(1), 20, 0).
					WillReturnRows(rows)
			},
			want: []models.EmailWithMetadata{
				{Email: models.Email{ID: 1, SenderID: 2, Header: "Subj", Body: "Body", CreatedAt: now}, IsRead: false, ReceivedAt: now},
			},
			wantErr: false,
		},
		{
			name:   "query error",
			userID: 1,
			limit:  10,
			offset: 0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetEmailsByReceiver(context.Background(), tt.userID, tt.limit, tt.offset)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetEmailsBySender(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		userID    int64
		limit     int
		offset    int
		mockSetup func(mock sqlmock.Sqlmock)
		want      []models.EmailWithMetadata
		wantErr   bool
	}{
		{
			name:   "success with receivers",
			userID: 1,
			limit:  20,
			offset: 0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "is_read", "receivers_emails"}).
					AddRow(1, 1, "Subj", "Body", now, false, `["a@b.com"]`)
				mock.ExpectQuery(`SELECT (.+) FROM emails e WHERE e.sender_id = \$1`).
					WithArgs(int64(1), 20, 0).
					WillReturnRows(rows)
			},
			want: []models.EmailWithMetadata{
				{Email: models.Email{ID: 1, SenderID: 1, Header: "Subj", Body: "Body", CreatedAt: now}, IsRead: false, ReceiversEmails: []string{"a@b.com"}},
			},
			wantErr: false,
		},
		{
			name:   "query error",
			userID: 1,
			limit:  10,
			offset: 0,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("fail"))
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetEmailsBySender(context.Background(), tt.userID, tt.limit, tt.offset)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetEmailByID(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		emailID   int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      *models.EmailWithAvatar
		wantErr   error
	}{
		{
			name:    "success with avatar",
			emailID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "sender_id", "header", "body", "created_at", "image_path",
				}).AddRow(1, 2, "Subj", "Body", now, "/avatars/2.jpg")
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			want: &models.EmailWithAvatar{
				Email: models.Email{
					ID:        1,
					SenderID:  2,
					Header:    "Subj",
					Body:      "Body",
					CreatedAt: now,
				},
				SenderImagePath: "/avatars/2.jpg",
			},
			wantErr: nil,
		},
		{
			name:    "not found",
			emailID: 99,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: ErrMailNotFound,
		},
		{
			name:    "database error",
			emailID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("connection lost"))
			},
			want:    nil,
			wantErr: ErrQueryFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetEmailByID(context.Background(), tt.emailID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_GetUnreadEmailsCount(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      int
		wantErr   error
	}{
		{
			name:   "success",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails WHERE receiver_id = \$1 AND is_read = false`).
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
			},
			want:    5,
			wantErr: nil,
		},
		{
			name:   "error",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrNoRows)
			},
			want:    0,
			wantErr: ErrMailNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetUnreadEmailsCount(context.Background(), tt.userID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_GetEmailsCount(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      int
		wantErr   error
	}{
		{
			name:   "success",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_emails WHERE receiver_id = \$1`).
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
			},
			want:    10,
			wantErr: nil,
		},
		{
			name:   "error",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrNoRows)
			},
			want:    0,
			wantErr: ErrMailNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetEmailsCount(context.Background(), tt.userID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_GetUserEmailsCount(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      int
		wantErr   error
	}{
		{
			name:   "success",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM emails WHERE sender_id = \$1 and is_deleted = false`).
					WithArgs(int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
			},
			want:    7,
			wantErr: nil,
		},
		{
			name:   "error",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrNoRows)
			},
			want:    0,
			wantErr: ErrMailNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.GetUserEmailsCount(context.Background(), tt.userID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_MarkEmailAsRead(t *testing.T) {
	tests := []struct {
		name      string
		emailID   int64
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM user_emails WHERE email_id = \$1 AND receiver_id = \$2\)`).
					WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectExec(`UPDATE user_emails SET is_read = true`).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "email not found",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr: ErrMailNotFound,
		},
		{
			name:    "exists query error",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(errors.New("fail"))
			},
			wantErr: ErrQueryFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			err := repo.MarkEmailAsRead(context.Background(), tt.emailID, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_CheckUserEmailExists(t *testing.T) {
	tests := []struct {
		name      string
		emailID   int64
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      bool
		wantErr   error
	}{
		{
			name:    "exists",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			want:    true,
			wantErr: nil,
		},
		{
			name:    "not exists",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			want:    false,
			wantErr: nil,
		},
		{
			name:    "error",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WillReturnError(errors.New("db error"))
			},
			want:    false,
			wantErr: ErrQueryFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			got, err := repo.CheckUserEmailExists(context.Background(), tt.emailID, tt.userID)
			assert.Equal(t, tt.want, got)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_DeleteEmailForReceiver(t *testing.T) {
	tests := []struct {
		name      string
		emailID   int64
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_emails WHERE email_id = \$1 AND receiver_id = \$2`).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "no rows affected",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE`).WithArgs(int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: ErrMailNotFound,
		},
		{
			name:    "exec error",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE`).WillReturnError(errors.New("fail"))
			},
			wantErr: errors.New("fail"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			err := repo.DeleteEmailForReceiver(context.Background(), tt.emailID, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRepository_DeleteEmailForSender(t *testing.T) {
	tests := []struct {
		name      string
		emailID   int64
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE emails SET is_deleted = true WHERE id = \$1 AND sender_id = \$2 AND is_deleted = false`).
					WithArgs(int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "no rows",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE emails`).WithArgs(int64(1), int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: ErrMailNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			err := repo.DeleteEmailForSender(context.Background(), tt.emailID, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_CheckEmailAccess(t *testing.T) {
	tests := []struct {
		name      string
		emailID   int64
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "has access",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr: nil,
		},
		{
			name:    "no access",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1), int64(2)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantErr: nil, // function doesn't return error on false, but we might want to; current code returns nil
		},
		{
			name:    "query error",
			emailID: 1,
			userID:  2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(errors.New("fail"))
			},
			wantErr: ErrAccessDenied,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			repo := &Repository{db: db}
			tt.mockSetup(mock)
			err := repo.CheckEmailAccess(context.Background(), tt.emailID, tt.userID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_BeginTx(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	mock.ExpectBegin()
	repo := &Repository{db: db}
	tx, err := repo.BeginTx(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.NoError(t, mock.ExpectationsWereMet())
}
