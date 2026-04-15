package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_GetEmailsByReceiver(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		input         GetEmailsInput
		setupMock     func(*mocks.MockRepository)
		expected      *GetEmailsResult
		expectedError error
	}{
		{
			name: "success",
			input: GetEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123), 20, 0).
					Return([]models.EmailWithMetadata{
						{
							Email: models.Email{
								ID:        1,
								SenderID:  100,
								Header:    "Subject",
								Body:      "Body",
								CreatedAt: now,
							},
							IsRead:     false,
							ReceivedAt: now,
						},
					}, nil)
				m.EXPECT().
					GetEmailsCount(gomock.Any(), int64(123)).
					Return(10, nil)
				m.EXPECT().
					GetUnreadEmailsCount(gomock.Any(), int64(123)).
					Return(3, nil)
			},
			expected: &GetEmailsResult{
				Emails: []EmailResult{
					{
						ID:        1,
						SenderID:  100,
						Header:    "Subject",
						Body:      "Body",
						CreatedAt: now,
						IsRead:    false,
					},
				},
				Limit:       20,
				Offset:      0,
				Total:       10,
				UnreadCount: 3,
			},
			expectedError: nil,
		},
		{
			name: "repository GetEmailsByReceiver error",
			input: GetEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123), 20, 0).
					Return(nil, repository.ErrQueryFail)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "repository GetEmailsCount error",
			input: GetEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123), 20, 0).
					Return([]models.EmailWithMetadata{}, nil)
				m.EXPECT().
					GetEmailsCount(gomock.Any(), int64(123)).
					Return(0, repository.ErrQueryFail)
				m.EXPECT().
					GetUnreadEmailsCount(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "repository GetUnreadEmailsCount error",
			input: GetEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123), 20, 0).
					Return([]models.EmailWithMetadata{}, nil)
				m.EXPECT().
					GetEmailsCount(gomock.Any(), int64(123)).
					Return(10, nil)
				m.EXPECT().
					GetUnreadEmailsCount(gomock.Any(), int64(123)).
					Return(0, repository.ErrQueryFail)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			result, err := s.GetEmailsByReceiver(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestService_GetEmailsBySender(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		input         GetMyEmailsInput
		setupMock     func(*mocks.MockRepository)
		expected      *GetMyEmailsResult
		expectedError error
	}{
		{
			name: "success",
			input: GetMyEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), int64(123), 20, 0).
					Return([]models.EmailWithMetadata{
						{
							Email: models.Email{
								ID:        1,
								SenderID:  123,
								Header:    "Sent Mail",
								Body:      "Content",
								CreatedAt: now,
							},
							IsRead:          false,
							ReceiversEmails: []string{"receiver@example.com"},
						},
					}, nil)
				m.EXPECT().
					GetUserEmailsCount(gomock.Any(), int64(123)).
					Return(5, nil)
			},
			expected: &GetMyEmailsResult{
				Emails: []MyEmailResult{
					{
						ID:              1,
						SenderID:        123,
						Header:          "Sent Mail",
						Body:            "Content",
						CreatedAt:       now,
						IsRead:          false,
						ReceiversEmails: []string{"receiver@example.com"},
					},
				},
				Limit:  20,
				Offset: 0,
				Total:  5,
			},
			expectedError: nil,
		},
		{
			name: "repository GetEmailsBySender error",
			input: GetMyEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), int64(123), 20, 0).
					Return(nil, repository.ErrQueryFail)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "repository GetUserEmailsCount error",
			input: GetMyEmailsInput{
				UserID: 123,
				Limit:  20,
				Offset: 0,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), int64(123), 20, 0).
					Return([]models.EmailWithMetadata{}, nil)
				m.EXPECT().
					GetUserEmailsCount(gomock.Any(), int64(123)).
					Return(0, repository.ErrQueryFail)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			result, err := s.GetEmailsBySender(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestService_GetEmailByID(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name          string
		input         GetEmailInput
		setupMock     func(*mocks.MockRepository)
		expected      *GetEmailResult
		expectedError error
	}{
		{
			name: "success",
			input: GetEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckEmailAccess(gomock.Any(), int64(1), int64(123)).
					Return(nil)
				m.EXPECT().
					GetEmailByID(gomock.Any(), int64(1)).
					Return(&models.EmailWithAvatar{
						Email: models.Email{
							ID:        1,
							SenderID:  100,
							Header:    "Subject",
							Body:      "Body",
							CreatedAt: now,
						},
						SenderImagePath: "/avatars/100.jpg",
					}, nil)
			},
			expected: &GetEmailResult{
				ID:              1,
				SenderID:        100,
				Header:          "Subject",
				Body:            "Body",
				CreatedAt:       now,
				SenderImagePath: "/avatars/100.jpg",
			},
			expectedError: nil,
		},
		{
			name: "access denied",
			input: GetEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckEmailAccess(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrAccessDenied)
				m.EXPECT().
					GetEmailByID(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expected:      nil,
			expectedError: ErrAccessDenied,
		},
		{
			name: "GetEmailByID error",
			input: GetEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckEmailAccess(gomock.Any(), int64(1), int64(123)).
					Return(nil)
				m.EXPECT().
					GetEmailByID(gomock.Any(), int64(1)).
					Return(nil, repository.ErrMailNotFound)
			},
			expected:      nil,
			expectedError: ErrEmailNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			result, err := s.GetEmailByID(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestService_DeleteEmailForReceiver(t *testing.T) {
	tests := []struct {
		name          string
		input         DeleteEmailInput
		setupMock     func(*mocks.MockRepository)
		expectedError error
	}{
		{
			name: "success",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckUserEmailExists(gomock.Any(), int64(1), int64(123)).
					Return(true, nil)
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), int64(1), int64(123)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "email not found",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckUserEmailExists(gomock.Any(), int64(1), int64(123)).
					Return(false, nil)
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedError: ErrEmailNotFound,
		},
		{
			name: "CheckUserEmailExists error",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckUserEmailExists(gomock.Any(), int64(1), int64(123)).
					Return(false, repository.ErrQueryFail)
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "DeleteEmailForReceiver error",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					CheckUserEmailExists(gomock.Any(), int64(1), int64(123)).
					Return(true, nil)
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrMailNotFound)
			},
			expectedError: ErrEmailNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			err := s.DeleteEmailForReceiver(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_DeleteEmailForSender(t *testing.T) {
	tests := []struct {
		name          string
		input         DeleteEmailInput
		setupMock     func(*mocks.MockRepository)
		expectedError error
	}{
		{
			name: "success",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), int64(1), int64(123)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository error",
			input: DeleteEmailInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrMailNotFound)
			},
			expectedError: ErrEmailNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			err := s.DeleteEmailForSender(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_MarkEmailAsRead(t *testing.T) {
	tests := []struct {
		name          string
		input         MarkAsReadInput
		setupMock     func(*mocks.MockRepository)
		expectedError error
	}{
		{
			name: "success",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), int64(1), int64(123)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "repository error - not found",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrMailNotFound)
			},
			expectedError: ErrEmailNotFound,
		},
		{
			name: "repository error - query fail",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrQueryFail)
			},
			expectedError: repository.ErrQueryFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			err := s.MarkEmailAsRead(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_SendEmail_IntegrationWithSQLMock(t *testing.T) {
	tests := []struct {
		name          string
		input         SendEmailInput
		mockSetup     func(mock sqlmock.Sqlmock)
		expected      *SendEmailResult
		expectedError error
	}{
		{
			name: "successful send",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("receiver@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "receiver@smail.ru", "Rec", "User"))

				mock.ExpectBegin()

				mock.ExpectQuery(`INSERT INTO emails \(sender_id, header, body\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
					WithArgs(int64(123), "Hello", "World").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

				mock.ExpectExec(`INSERT INTO user_emails \(receiver_id, email_id\) VALUES \(\$1, \$2\)`).
					WithArgs(int64(456), int64(10)).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			expected: &SendEmailResult{
				ID:        10,
				SenderID:  123,
				Header:    "Hello",
				Body:      "World",
				CreatedAt: time.Time{},
			},
			expectedError: nil,
		},
		{
			name: "resolveReceivers fails - empty receivers",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{},
			},
			mockSetup:     func(mock sqlmock.Sqlmock) {},
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "resolveReceivers fails - no users found",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"unknown@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("unknown@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}))
			},
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "resolveReceivers - GetUsersByEmails error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
			},
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "BeginTx error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users`).
					WithArgs("receiver@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "receiver@smail.ru", "Rec", "User"))
				mock.ExpectBegin().WillReturnError(errors.New("tx error"))
			},
			expectedError: ErrTransaction,
		},
		{
			name: "SaveEmailWithTx error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users`).
					WithArgs("receiver@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "receiver@smail.ru", "Rec", "User"))
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Hello", "World").
					WillReturnError(errors.New("insert failed"))
				mock.ExpectRollback()
			},
			expectedError: repository.ErrSaveEmail,
		},
		{
			name: "AddEmailReceiversWithTx error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users`).
					WithArgs("receiver@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "receiver@smail.ru", "Rec", "User"))
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Hello", "World").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(456), int64(10)).
					WillReturnError(errors.New("add receivers failed"))
				mock.ExpectRollback()
			},
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "Commit error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"receiver@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users`).
					WithArgs("receiver@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "receiver@smail.ru", "Rec", "User"))
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Hello", "World").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(456), int64(10)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			expectedError: ErrTransaction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := repository.New(db)
			tt.mockSetup(mock)

			s := New(repo)
			result, err := s.SendEmail(context.Background(), tt.input)

			if tt.expectedError != nil {
				require.Error(t, err)
				if errors.Is(tt.expectedError, ErrNoValidReceivers) || errors.Is(tt.expectedError, ErrTransaction) || errors.Is(tt.expectedError, repository.ErrSaveEmail) {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.EqualError(t, err, tt.expectedError.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.SenderID, result.SenderID)
				assert.Equal(t, tt.expected.Header, result.Header)
				assert.Equal(t, tt.expected.Body, result.Body)
				// CreatedAt is zero because it is not returned from the DB
				assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestService_resolveReceivers(t *testing.T) {
	tests := []struct {
		name          string
		emails        []string
		setupMock     func(*mocks.MockRepository)
		expectedIDs   []int64
		expectedError error
	}{
		{
			name:   "success",
			emails: []string{"a@b.com", "c@d.com"},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetUsersByEmails(gomock.Any(), []string{"a@b.com", "c@d.com"}).
					Return([]*models.User{
						{ID: 1, Email: "a@b.com"},
						{ID: 2, Email: "c@d.com"},
					}, nil)
			},
			expectedIDs:   []int64{1, 2},
			expectedError: nil,
		},
		{
			name:          "empty emails",
			emails:        []string{},
			setupMock:     func(m *mocks.MockRepository) {},
			expectedIDs:   nil,
			expectedError: ErrNoValidReceivers,
		},
		{
			name:   "repository error",
			emails: []string{"a@b.com"},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetUsersByEmails(gomock.Any(), gomock.Any()).
					Return(nil, repository.ErrQueryFail)
			},
			expectedIDs:   nil,
			expectedError: repository.ErrQueryFail,
		},
		{
			name:   "no users found",
			emails: []string{"unknown@b.com"},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetUsersByEmails(gomock.Any(), gomock.Any()).
					Return([]*models.User{}, nil)
			},
			expectedIDs:   nil,
			expectedError: ErrNoValidReceivers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			ids, err := s.resolveReceivers(context.Background(), tt.emails)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIDs, ids)
			}
		})
	}
}

func Test_mapRepositoryError(t *testing.T) {
	otherErr := errors.New("some other error")
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{"ErrDuplicate -> ErrConflict", repository.ErrDuplicate, ErrConflict},
		{"ErrForeignKey -> ErrBadRequest", repository.ErrForeignKey, ErrBadRequest},
		{"ErrUserNotFound -> ErrUserNotFound", repository.ErrUserNotFound, ErrUserNotFound},
		{"ErrReceiverAdd -> ErrNoValidReceivers", repository.ErrReceiverAdd, ErrNoValidReceivers},
		{"ErrMailNotFound -> ErrEmailNotFound", repository.ErrMailNotFound, ErrEmailNotFound},
		{"ErrAccessDenied -> ErrAccessDenied", repository.ErrAccessDenied, ErrAccessDenied},
		{"unknown error unchanged", otherErr, otherErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRepositoryError(tt.err)
			assert.True(t, errors.Is(result, tt.expected))
		})
	}
}

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *repository.Repository) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	repo := repository.New(db)
	return db, mock, repo
}

func TestService_ForwardEmail_IntegrationWithSQLMock(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		input         ForwardEmailInput
		mockSetup     func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "successful forward",
			input: ForwardEmailInput{
				UserID:    123,
				EmailID:   10,
				Receivers: []string{"friend@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path"}).
						AddRow(10, 100, "Original Subject", "Original Body", now, "/avatar.jpg"))

				mock.ExpectQuery(`INSERT INTO emails \(sender_id, header, body\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
					WithArgs(int64(123), "Original Subject", "Original Body").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))

				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("friend@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "friend@smail.ru", "Friend", "User"))

				mock.ExpectExec(`INSERT INTO user_emails \(receiver_id, email_id\) VALUES \(\$1, \$2\)`).
					WithArgs(int64(456), int64(20)).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name: "BeginTx error",
			input: ForwardEmailInput{
				UserID:  123,
				EmailID: 10,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("connection lost"))
			},
			expectedError: ErrTransaction,
		},
		{
			name: "CheckEmailAccess denied",
			input: ForwardEmailInput{
				UserID:  123,
				EmailID: 10,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnError(repository.ErrAccessDenied)
				mock.ExpectRollback()
			},
			expectedError: ErrAccessDenied,
		},
		{
			name: "GetEmailByID not found",
			input: ForwardEmailInput{
				UserID:  123,
				EmailID: 999,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(999), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			expectedError: ErrEmailNotFound,
		},
		{
			name: "SaveEmailWithTx error",
			input: ForwardEmailInput{
				UserID:    123,
				EmailID:   10,
				Receivers: []string{"friend@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// Access check
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				// GetEmailByID
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path"}).
						AddRow(10, 100, "Original Subject", "Original Body", now, "/avatar.jpg"))
				// SaveEmailWithTx fails
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Original Subject", "Original Body").
					WillReturnError(errors.New("insert failed"))
				mock.ExpectRollback()
			},
			expectedError: repository.ErrSaveEmail,
		},
		{
			name: "resolveReceivers no valid receivers",
			input: ForwardEmailInput{
				UserID:    123,
				EmailID:   10,
				Receivers: []string{"unknown@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path"}).
						AddRow(10, 100, "Original Subject", "Original Body", now, "/avatar.jpg"))
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Original Subject", "Original Body").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("unknown@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}))
				mock.ExpectRollback()
			},
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "AddEmailReceiversWithTx error",
			input: ForwardEmailInput{
				UserID:    123,
				EmailID:   10,
				Receivers: []string{"friend@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path"}).
						AddRow(10, 100, "Original Subject", "Original Body", now, "/avatar.jpg"))
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Original Subject", "Original Body").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("friend@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "friend@smail.ru", "Friend", "User"))
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(456), int64(20)).
					WillReturnError(errors.New("foreign key violation"))
				mock.ExpectRollback()
			},
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "Commit error",
			input: ForwardEmailInput{
				UserID:    123,
				EmailID:   10,
				Receivers: []string{"friend@smail.ru"},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT EXISTS\(`).
					WithArgs(int64(10), int64(123)).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(`SELECT (.+) FROM emails e JOIN users u`).
					WithArgs(int64(10)).
					WillReturnRows(sqlmock.NewRows([]string{"id", "sender_id", "header", "body", "created_at", "image_path"}).
						AddRow(10, 100, "Original Subject", "Original Body", now, "/avatar.jpg"))
				mock.ExpectQuery(`INSERT INTO emails`).
					WithArgs(int64(123), "Original Subject", "Original Body").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))
				mock.ExpectQuery(`SELECT id, email, name, surname FROM users WHERE email IN \(\$1\)`).
					WithArgs("friend@smail.ru").
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "surname"}).
						AddRow(456, "friend@smail.ru", "Friend", "User"))
				mock.ExpectExec(`INSERT INTO user_emails`).
					WithArgs(int64(456), int64(20)).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			expectedError: ErrTransaction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, repo := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			s := New(repo)
			err := s.ForwardEmail(context.Background(), tt.input)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestService_MarkEmailAsUnRead(t *testing.T) {
	tests := []struct {
		name          string
		input         MarkAsReadInput
		setupMock     func(*mocks.MockRepository)
		expectedError error
	}{
		{
			name: "success",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsUnRead(gomock.Any(), int64(1), int64(123)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "email not found",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 999,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsUnRead(gomock.Any(), int64(999), int64(123)).
					Return(repository.ErrMailNotFound)
			},
			expectedError: ErrEmailNotFound,
		},
		{
			name: "repository error - query fail",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsUnRead(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrQueryFail)
			},
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "repository error - access denied",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					MarkEmailAsUnRead(gomock.Any(), int64(1), int64(123)).
					Return(repository.ErrAccessDenied)
			},
			expectedError: ErrAccessDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			tt.setupMock(mockRepo)

			s := New(mockRepo)
			err := s.MarkEmailAsUnRead(context.Background(), tt.input)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
