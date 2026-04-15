package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/delivery/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/gorilla/mux"
	"go.uber.org/mock/gomock"
)

func TestHandler_GetEmails(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		userID           int64
		setupMock        func(*mocks.MockService)
		expectedStatus   int
		expectedResponse *GetEmailsResponse
	}{
		{
			name:   "successful get emails",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				expectedEmails := &service.GetEmailsResult{
					Emails: []service.EmailResult{
						{
							ID:        1,
							SenderID:  100,
							Header:    "Welcome to PushToMain",
							Body:      "Hello! Welcome to our platform.",
							CreatedAt: now,
							IsRead:    false,
						},
						{
							ID:        2,
							SenderID:  200,
							Header:    "Your weekly digest",
							Body:      "Here's what you missed this week.",
							CreatedAt: now.Add(-24 * time.Hour),
							IsRead:    true,
						},
					},
					Limit:       20,
					Offset:      0,
					Total:       2,
					UnreadCount: 1,
				}
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
						UserID: 123,
						Limit:  20,
						Offset: 0,
					}).
					Return(expectedEmails, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetEmailsResponse{
				Emails: []EmailResponse{
					{
						ID:        1,
						SenderID:  100,
						Header:    "Welcome to PushToMain",
						Body:      "Hello! Welcome to our platform.",
						CreatedAt: now,
						IsRead:    false,
					},
					{
						ID:        2,
						SenderID:  200,
						Header:    "Your weekly digest",
						Body:      "Here's what you missed this week.",
						CreatedAt: now.Add(-24 * time.Hour),
						IsRead:    true,
					},
				},
				Limit:       20,
				Offset:      0,
				Total:       2,
				UnreadCount: 1,
			},
		},
		{
			name:   "empty emails list",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{UserID: 123, Limit: 20, Offset: 0}).
					Return(&service.GetEmailsResult{
						Emails:      []service.EmailResult{},
						Limit:       20,
						Offset:      0,
						Total:       0,
						UnreadCount: 0,
					},
						nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetEmailsResponse{
				Emails:      []EmailResponse{},
				Limit:       20,
				Offset:      0,
				Total:       0,
				UnreadCount: 0,
			},
		},
		{
			name:   "invalid user id - zero",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: nil,
		},
		{
			name:   "invalid user id - negative",
			userID: -5,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: nil,
		},
		{
			name:   "service returns error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{UserID: 123, Limit: 20, Offset: 0}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: nil,
		},
		{
			name:   "service returns database error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{UserID: 123, Limit: 20, Offset: 0}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{
				service: mockService,
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/emails", nil)

			payload := &utils.JwtPayload{
				UserId: tt.userID,
				Exp:    time.Now().Add(time.Hour).Unix(),
			}
			ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.GetEmails(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedResponse != nil {
				var response GetEmailsResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response body: %v", err)
				}

				if len(response.Emails) != len(tt.expectedResponse.Emails) {
					t.Errorf("expected %d emails, got %d", len(tt.expectedResponse.Emails), len(response.Emails))
				}

				for i, email := range response.Emails {
					if i >= len(tt.expectedResponse.Emails) {
						break
					}
					expected := tt.expectedResponse.Emails[i]

					if email.ID != expected.ID {
						t.Errorf("email %d: expected ID %d, got %d", i, expected.ID, email.ID)
					}
					if email.SenderID != expected.SenderID {
						t.Errorf("email %d: expected SenderID %d, got %d", i, expected.SenderID, email.SenderID)
					}
					if email.Header != expected.Header {
						t.Errorf("email %d: expected Header %q, got %q", i, expected.Header, email.Header)
					}
					if email.Body != expected.Body {
						t.Errorf("email %d: expected Body %q, got %q", i, expected.Body, email.Body)
					}
				}
			}
		})
	}
}

func TestHandler_GetEmails_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().
		GetEmailsByReceiver(gomock.Any(), gomock.Any()).
		Times(0)

	handler := &Handler{
		service: mockService,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails", nil)
	w := httptest.NewRecorder()

	handler.GetEmails(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_GetEmails_InvalidClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().
		GetEmailsByReceiver(gomock.Any(), gomock.Any()).
		Times(0)

	handler := &Handler{
		service: mockService,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails", nil)

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, &utils.JwtPayload{
		UserId: -1,
		Exp:    time.Now().Add(time.Hour).Unix(),
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetEmails(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_GetEmails_ManyEmails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)

	expectedResponse := &service.GetEmailsResult{
		Emails:      make([]service.EmailResult, 1000),
		Limit:       20,
		Offset:      0,
		Total:       1000,
		UnreadCount: 1000,
	}

	for i := 0; i < 1000; i++ {
		expectedResponse.Emails[i] = service.EmailResult{
			ID:        int64(i + 1),
			SenderID:  int64(100 + i%10),
			Header:    "Email " + string(rune(i)),
			Body:      "Body of email " + string(rune(i)),
			CreatedAt: time.Now(),
		}
	}

	mockService.EXPECT().
		GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
			UserID: 123,
			Limit:  20,
			Offset: 0,
		}).Return(expectedResponse, nil)

	handler := &Handler{
		service: mockService,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails", nil)
	ctx := context.WithValue(req.Context(),
		middleware.ClaimsKey,
		&utils.JwtPayload{
			UserId: 123,
			Exp:    time.Now().Add(time.Hour).Unix(),
		},
	)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetEmails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response *service.GetEmailsResult
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if len(response.Emails) != 1000 {
		t.Errorf("expected 1000 emails, got %d", len(response.Emails))
	}

	if response.Limit != 20 {
		t.Errorf("expected limit 20, got %d", response.Limit)
	}
}

func TestHandler_GetMyEmails(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		userID           int64
		setupMock        func(*mocks.MockService)
		expectedStatus   int
		expectedResponse *GetMyEmailsResponse
	}{
		{
			name:   "successful get sent emails",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				expectedEmails := &service.GetMyEmailsResult{
					Emails: []service.MyEmailResult{
						{
							ID:        1,
							SenderID:  456,
							Header:    "Project update",
							Body:      "Here's the latest.",
							CreatedAt: now,
							IsRead:    false,
							ReceiversEmails: []string{
								"foo@smail.ru",
								"bar@smail.ru",
							},
						},
						{
							ID:        2,
							SenderID:  789,
							Header:    "Meeting notes",
							Body:      "Summary attached.",
							CreatedAt: now.Add(-48 * time.Hour),
							IsRead:    true,
							ReceiversEmails: []string{
								"foo@smail.ru",
							},
						},
					},
					Limit:  20,
					Offset: 0,
					Total:  2,
				}
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), service.GetMyEmailsInput{
						UserID: 123,
						Limit:  20,
						Offset: 0,
					}).
					Return(expectedEmails, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetMyEmailsResponse{
				Emails: []MyEmailResponse{
					{
						ID:              1,
						SenderID:        456,
						Header:          "Project update",
						Body:            "Here's the latest.",
						CreatedAt:       now,
						IsRead:          false,
						ReceiversEmails: []string{"foo@smail.ru", "bar@smail.ru"},
					},
					{
						ID:              2,
						SenderID:        789,
						Header:          "Meeting notes",
						Body:            "Summary attached.",
						CreatedAt:       now.Add(-48 * time.Hour),
						IsRead:          true,
						ReceiversEmails: []string{"foo@smail.ru"},
					},
				},
				Limit:  20,
				Offset: 0,
				Total:  2,
			},
		},
		{
			name:   "empty sent list",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), service.GetMyEmailsInput{
						UserID: 123,
						Limit:  20,
						Offset: 0,
					}).
					Return(&service.GetMyEmailsResult{
						Emails: []service.MyEmailResult{},
						Limit:  20,
						Offset: 0,
						Total:  0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetMyEmailsResponse{
				Emails: []MyEmailResponse{},
				Limit:  20,
				Offset: 0,
				Total:  0,
			},
		},
		{
			name:   "invalid user id - zero",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: nil,
		},
		{
			name:   "service error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), gomock.Any()).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/emails/sent", nil)
			payload := &utils.JwtPayload{UserId: tt.userID, Exp: time.Now().Add(time.Hour).Unix()}
			ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.GetMyEmails(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedResponse != nil {
				var resp GetMyEmailsResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(resp.Emails) != len(tt.expectedResponse.Emails) {
					t.Errorf("expected %d emails, got %d", len(tt.expectedResponse.Emails), len(resp.Emails))
				}
			}
		})
	}
}

func TestHandler_GetMyEmails_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().GetEmailsBySender(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/myemails", nil)
	w := httptest.NewRecorder()

	handler.GetMyEmails(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_SendEmail(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		requestBody      interface{}
		setupMock        func(m *mocks.MockService)
		expectedStatus   int
		expectedResponse *SendEmailResponse
	}{
		{
			name: "successful send email",
			requestBody: SendEmailRequest{
				Header:    "test subject",
				Body:      "test message",
				Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SendEmail(gomock.Any(), service.SendEmailInput{
						UserId:    123,
						Header:    "test subject",
						Body:      "test message",
						Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
					}).
					Return(&service.SendEmailResult{
						ID:        1,
						SenderID:  123,
						Header:    "test subject",
						Body:      "test message",
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &SendEmailResponse{
				ID:        1,
				SenderID:  123,
				Header:    "test subject",
				Body:      "test message",
				CreatedAt: now,
			},
		},
		{
			name: "service error",
			requestBody: SendEmailRequest{
				Header:    "test subject",
				Body:      "test message",
				Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SendEmail(gomock.Any(), service.SendEmailInput{
						UserId:    123,
						Header:    "test subject",
						Body:      "test message",
						Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
					}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "missing claims",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			var body bytes.Buffer
			if tt.requestBody != nil {
				if err := json.NewEncoder(&body).Encode(tt.requestBody); err != nil {
					t.Fatalf("failed to encode request body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/send", &body)
			if tt.name != "missing claims" {
				payload := &utils.JwtPayload{UserId: 123, Exp: time.Now().Add(time.Hour).Unix()}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.SendEmail(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedResponse != nil {
				var resp SendEmailResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.ID != tt.expectedResponse.ID {
					t.Errorf("expected ID %d, got %d", tt.expectedResponse.ID, resp.ID)
				}
				if resp.SenderID != tt.expectedResponse.SenderID {
					t.Errorf("expected SenderID %d, got %d", tt.expectedResponse.SenderID, resp.SenderID)
				}
				if resp.Header != tt.expectedResponse.Header {
					t.Errorf("expected Header %s, got %s", tt.expectedResponse.Header, resp.Header)
				}
				if resp.Body != tt.expectedResponse.Body {
					t.Errorf("expected Body %s, got %s", tt.expectedResponse.Body, resp.Body)
				}
				if !resp.CreatedAt.Equal(tt.expectedResponse.CreatedAt) {
					t.Errorf("expected CreatedAt %v, got %v", tt.expectedResponse.CreatedAt, resp.CreatedAt)
				}
			}
		})
	}
}

func TestHandler_GetEmailByID(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		emailID          string
		userID           int64
		skipClaims       bool
		setupMock        func(m *mocks.MockService)
		expectedStatus   int
		expectedResponse *GetEmailResponse
	}{
		{
			name:       "successful get email by id",
			emailID:    "1",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 1, UserID: 123}).
					Return(&service.GetEmailResult{
						ID:        1,
						SenderID:  456,
						Header:    "Project update",
						Body:      "Here's the latest.",
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetEmailResponse{
				ID:        1,
				SenderID:  456,
				Header:    "Project update",
				Body:      "Here's the latest.",
				CreatedAt: now,
			},
		},
		{
			name:       "email not found",
			emailID:    "999",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 999, UserID: 123}).
					Return(nil, service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "access denied",
			emailID:    "2",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 2, UserID: 123}).
					Return(nil, service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "service error",
			emailID:    "3",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 3, UserID: 123}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid email ID format",
			emailID:    "abc",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email ID",
			emailID:    "",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID - zero",
			emailID:    "1",
			userID:     0,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID - negative",
			emailID:    "1",
			userID:     -5,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			emailID:    "1",
			userID:     0,
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "large email ID",
			emailID:    "9223372036854775807",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 9223372036854775807, UserID: 123}).
					Return(&service.GetEmailResult{
						ID:        9223372036854775807,
						SenderID:  456,
						Header:    "Large ID email",
						Body:      "Content",
						CreatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedResponse: &GetEmailResponse{
				ID:        9223372036854775807,
				SenderID:  456,
				Header:    "Large ID email",
				Body:      "Content",
				CreatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			url := "/api/v1/emails/" + tt.emailID
			req := httptest.NewRequest(http.MethodGet, url, nil)

			if tt.emailID != "" {
				req = mux.SetURLVars(req, map[string]string{"id": tt.emailID})
			}

			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.GetEmailByID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedResponse != nil {
				var resp GetEmailResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.ID != tt.expectedResponse.ID {
					t.Errorf("expected ID %d, got %d", tt.expectedResponse.ID, resp.ID)
				}
				if resp.SenderID != tt.expectedResponse.SenderID {
					t.Errorf("expected SenderID %d, got %d", tt.expectedResponse.SenderID, resp.SenderID)
				}
				if resp.Header != tt.expectedResponse.Header {
					t.Errorf("expected Header %q, got %q", tt.expectedResponse.Header, resp.Header)
				}
				if resp.Body != tt.expectedResponse.Body {
					t.Errorf("expected Body %q, got %q", tt.expectedResponse.Body, resp.Body)
				}
				if !resp.CreatedAt.Equal(tt.expectedResponse.CreatedAt) {
					t.Errorf("expected CreatedAt %v, got %v", tt.expectedResponse.CreatedAt, resp.CreatedAt)
				}
			} else if tt.expectedStatus == http.StatusOK {
				var resp GetEmailResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("expected valid JSON response, got decode error: %v", err)
				}
			}
		})
	}
}

func TestHandler_GetEmailByID_WithQueryParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	mockService := mocks.NewMockService(ctrl)

	mockService.EXPECT().
		GetEmailByID(gomock.Any(), service.GetEmailInput{EmailID: 1, UserID: 123}).
		Return(&service.GetEmailResult{
			ID:        1,
			SenderID:  456,
			Header:    "Project update",
			Body:      "Here's the latest.",
			CreatedAt: now,
		}, nil)

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails/1?include_metadata=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	payload := &utils.JwtPayload{UserId: 123, Exp: time.Now().Add(time.Hour).Unix()}
	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetEmailByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp GetEmailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
}

func TestHandler_GetEmailByID_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetEmailByID(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_ForwardEmail(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "successful forward email",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), service.ForwardEmailInput{
						UserID:    123,
						EmailID:   5,
						Receivers: []string{"foo@smail.ru", "bar@smail.ru"},
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "successful forward with single receiver",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   10,
				Receivers: []string{"single@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), service.ForwardEmailInput{
						UserID:    123,
						EmailID:   10,
						Receivers: []string{"single@smail.ru"},
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "email not found",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   999,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), service.ForwardEmailInput{
						UserID:    123,
						EmailID:   999,
						Receivers: []string{"test@smail.ru"},
					}).
					Return(service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "access denied",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   2,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), service.ForwardEmailInput{
						UserID:    123,
						EmailID:   2,
						Receivers: []string{"test@smail.ru"},
					}).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "invalid receiver email format",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"not-an-email"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), service.ForwardEmailInput{
						UserID:    123,
						EmailID:   5,
						Receivers: []string{"not-an-email"},
					}).
					Return(service.ErrBadRequest)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "service internal error",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					ForwardEmail(gomock.Any(), gomock.Any()).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid request body - malformed JSON",
			userID:      123,
			skipClaims:  false,
			requestBody: `{"email_id": "not_a_number", "receivers": []}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid request body - empty receivers",
			userID:     123,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid request body - missing email_id",
			userID:     123,
			skipClaims: false,
			requestBody: map[string]interface{}{
				"receivers": []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			userID:     0,
			skipClaims: true,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID in claims - zero",
			userID:     0,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID in claims - negative",
			userID:     -1,
			skipClaims: false,
			requestBody: ForwardEmailRequest{
				EmailID:   5,
				Receivers: []string{"test@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty body",
			userID:      123,
			skipClaims:  false,
			requestBody: nil,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			var body bytes.Buffer
			if tt.requestBody != nil {
				switch v := tt.requestBody.(type) {
				case string:
					body.WriteString(v)
				default:
					if err := json.NewEncoder(&body).Encode(v); err != nil {
						t.Fatal(err)
					}
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/emails/forward", &body)
			req.Header.Set("Content-Type", "application/json")

			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.ForwardEmail(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_ForwardEmail_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().ForwardEmail(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	body := bytes.NewBufferString(`{"email_id": 5, "receivers": ["test@smail.ru"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/emails/forward", body)
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ForwardEmail(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_DeleteEmailForReceiver(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "successful delete",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 1,
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "email not found",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 999,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 999,
					}).
					Return(service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "access denied",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 2,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 2,
					}).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "service internal error",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 3,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 3,
					}).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid request body - malformed JSON",
			userID:      123,
			skipClaims:  false,
			requestBody: `{"email_id": "not_a_number"}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "invalid request body - missing email_id",
			userID:      123,
			skipClaims:  false,
			requestBody: map[string]interface{}{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email ID - zero",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 0,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email ID - negative",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: -5,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			userID:     0,
			skipClaims: true,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID in claims - zero",
			userID:     0,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID in claims - negative",
			userID:     -1,
			skipClaims: false,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty body",
			userID:      123,
			skipClaims:  false,
			requestBody: nil,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			var body bytes.Buffer
			if tt.requestBody != nil {
				switch v := tt.requestBody.(type) {
				case string:
					body.WriteString(v)
				default:
					if err := json.NewEncoder(&body).Encode(v); err != nil {
						t.Fatal(err)
					}
				}
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/emails", &body)
			req.Header.Set("Content-Type", "application/json")

			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.DeleteEmailForReceiver(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_DeleteEmailForReceiver_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	body := bytes.NewBufferString(`{"email_id": 1}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/emails", body)
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.DeleteEmailForReceiver(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_DeleteEmailForSender(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "successful delete for sender",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 1,
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "email not found",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 999,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 999,
					}).
					Return(service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "access denied",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 2,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 2,
					}).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "service internal error",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 3,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), service.DeleteEmailInput{
						UserID:  123,
						EmailID: 3,
					}).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid request body - malformed JSON",
			userID:      123,
			skipClaims:  false,
			requestBody: `{"email_id": "not_a_number"}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "invalid request body - missing email_id",
			userID:      123,
			skipClaims:  false,
			requestBody: map[string]interface{}{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email ID - zero",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 0,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email ID - negative",
			userID:     123,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: -5,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			userID:     0,
			skipClaims: true,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID in claims - zero",
			userID:     0,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID in claims - negative",
			userID:     -1,
			skipClaims: false,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty body",
			userID:      123,
			skipClaims:  false,
			requestBody: nil,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			var body bytes.Buffer
			if tt.requestBody != nil {
				switch v := tt.requestBody.(type) {
				case string:
					body.WriteString(v)
				default:
					if err := json.NewEncoder(&body).Encode(v); err != nil {
						t.Fatal(err)
					}
				}
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/myemails", &body)
			req.Header.Set("Content-Type", "application/json")

			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.DeleteEmailForSender(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_DeleteEmailForSender_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	body := bytes.NewBufferString(`{"email_id": 1}`)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/emails/sent", body)
	req.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.DeleteEmailForSender(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestHandler_MarkEmailAsRead(t *testing.T) {
	tests := []struct {
		name           string
		emailID        string
		userID         int64
		skipClaims     bool
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "successful mark as read",
			emailID:    "1",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
						UserID:  123,
						EmailID: 1,
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "email not found",
			emailID:    "999",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
						UserID:  123,
						EmailID: 999,
					}).
					Return(service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "access denied",
			emailID:    "2",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
						UserID:  123,
						EmailID: 2,
					}).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "service internal error",
			emailID:    "3",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
						UserID:  123,
						EmailID: 3,
					}).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid email ID format",
			emailID:    "abc",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email ID",
			emailID:    "",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			emailID:    "1",
			userID:     0,
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID - zero",
			emailID:    "1",
			userID:     0,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID - negative",
			emailID:    "1",
			userID:     -5,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "large email ID",
			emailID:    "9223372036854775807",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
						UserID:  123,
						EmailID: 9223372036854775807,
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			url := "/api/v1/emails/" + tt.emailID + "/read"
			req := httptest.NewRequest(http.MethodPut, url, nil)

			if tt.emailID != "" {
				req = mux.SetURLVars(req, map[string]string{"id": tt.emailID})
			}

			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.MarkEmailAsRead(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_MarkEmailAsRead_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/emails/1/read", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.MarkEmailAsRead(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
