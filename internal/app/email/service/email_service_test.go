package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	"github.com/stretchr/testify/assert"
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
					Return(&models.Email{
						ID:        1,
						SenderID:  100,
						Header:    "Subject",
						Body:      "Body",
						CreatedAt: now,
					}, nil)
			},
			expected: &GetEmailResult{
				ID:        1,
				SenderID:  100,
				Header:    "Subject",
				Body:      "Body",
				CreatedAt: now,
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
					GetEmailByID(gomock.Any(), int64(1)).
					Return(&models.Email{SenderID: 123}, nil)
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), int64(1), int64(123)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "access denied (user not sender)",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), int64(1)).
					Return(&models.Email{SenderID: 999}, nil)
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedError: ErrAccessDenied,
		},
		{
			name: "GetEmailByID error",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), int64(1)).
					Return(nil, repository.ErrMailNotFound)
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedError: ErrEmailNotFound,
		},
		{
			name: "MarkEmailAsRead error",
			input: MarkAsReadInput{
				UserID:  123,
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), int64(1)).
					Return(&models.Email{SenderID: 123}, nil)
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

func TestService_SendEmail(t *testing.T) {
	tests := []struct {
		name          string
		input         SendEmailInput
		setupMock     func(*mocks.MockRepository)
		expected      *SendEmailResult
		expectedError error
	}{
		{
			name: "resolveReceivers fails - no receivers",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{},
			},
			setupMock: func(m *mocks.MockRepository) {
			},
			expected:      nil,
			expectedError: ErrNoValidReceivers,
		},
		{
			name: "resolveReceivers - GetUsersByEmails error",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"bad@example.com"},
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetUsersByEmails(gomock.Any(), []string{"bad@example.com"}).
					Return(nil, repository.ErrQueryFail)
			},
			expected:      nil,
			expectedError: repository.ErrQueryFail,
		},
		{
			name: "resolveReceivers - no users found",
			input: SendEmailInput{
				UserId:    123,
				Header:    "Hello",
				Body:      "World",
				Receivers: []string{"unknown@example.com"},
			},
			setupMock: func(m *mocks.MockRepository) {
				m.EXPECT().
					GetUsersByEmails(gomock.Any(), []string{"unknown@example.com"}).
					Return([]*models.User{}, nil)
			},
			expected:      nil,
			expectedError: ErrNoValidReceivers,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockRepository(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mockRepo)
			}

			s := New(mockRepo)
			result, err := s.SendEmail(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Nil(t, result)
			} else {
				if tt.expected != nil {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, result)
				}
			}
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
