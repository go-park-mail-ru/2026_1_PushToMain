package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/delivery/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/models"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"go.uber.org/mock/gomock"
)

type contextKey string

var claimsKey contextKey = "claims"

func TestHandler_GetEmails(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		userID         int64
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedEmails *GetEmailsResponse
	}{
		{
			name:   "successful get emails",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				expectedEmails := []models.Email{
					{
						ID:        1,
						SenderID:  100,
						Header:    "Welcome to PushToMain",
						Body:      "Hello! Welcome to our platform.",
						CreatedAt: now,
					},
					{
						ID:        2,
						SenderID:  200,
						Header:    "Your weekly digest",
						Body:      "Here's what you missed this week.",
						CreatedAt: now.Add(-24 * time.Hour),
					},
				}
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), ).
					Return(expectedEmails, nil)
			},
			expectedStatus: http.StatusOK,
			expectedEmails: []models.Email{
				{
					ID:        1,
					SenderID:  100,
					Header:    "Welcome to PushToMain",
					Body:      "Hello! Welcome to our platform.",
					CreatedAt: now,
				},
				{
					ID:        2,
					SenderID:  200,
					Header:    "Your weekly digest",
					Body:      "Here's what you missed this week.",
					CreatedAt: now.Add(-24 * time.Hour),
				},
			},
		},
		{
			name:   "empty emails list",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123)).
					Return([]models.Email{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedEmails: []models.Email{},
		},
		{
			name:   "invalid user id - zero",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatus: http.StatusBadRequest,
			expectedEmails: nil,
		},
		{
			name:   "invalid user id - negative",
			userID: -5,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), gomock.Any()).
					Times(0)
			},
			expectedStatus: http.StatusBadRequest,
			expectedEmails: nil,
		},
		{
			name:   "service returns error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123)).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedEmails: nil,
		},
		{
			name:   "service returns database error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), int64(123)).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedEmails: nil,
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

			payload := &utils.JwtPayload{UserId: tt.userID, Exp: time.Now().Add(time.Hour).Unix()}
			ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handler.GetEmails(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedEmails != nil {
				var response []models.Email
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("failed to decode response body: %v", err)
				}

				if len(response) != len(tt.expectedEmails) {
					t.Errorf("expected %d emails, got %d", len(tt.expectedEmails), len(response))
				}

				for i, email := range response {
					if i >= len(tt.expectedEmails) {
						break
					}
					expected := tt.expectedEmails[i]

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

	ctx := context.WithValue(req.Context(), claimsKey, 0)
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

	expectedEmails := make([]models.Email, 1000)
	for i := 0; i < 1000; i++ {
		expectedEmails[i] = models.Email{
			ID:        int64(i + 1),
			SenderID:  int64(100 + i%10),
			Header:    "Email " + string(rune(i)),
			Body:      "Body of email " + string(rune(i)),
			CreatedAt: time.Now(),
		}
	}

	mockService.EXPECT().
		GetEmailsByReceiver(gomock.Any(), int64(123)).
		Return(expectedEmails, nil)

	handler := &Handler{
		service: mockService,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/emails", nil)
	ctx := context.WithValue(req.Context(), claimsKey, 123)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.GetEmails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []models.Email
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if len(response) != 1000 {
		t.Errorf("expected 1000 emails, got %d", len(response))
	}
}
