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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_GetEmails(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		query          string
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:   "success default pagination",
			userID: 1,
			query:  "",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
						UserID: 1,
						Limit:  20,
						Offset: 0,
					}).
					Return(&service.GetEmailsResult{
						Emails: []service.EmailResult{
							{ID: 1, SenderID: 2, Header: "h", Body: "b", CreatedAt: now, IsRead: false},
						},
						Limit:       20,
						Offset:      0,
						Total:       1,
						UnreadCount: 1,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:       "missing claims",
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid user id",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "custom pagination",
			userID: 1,
			query:  "?limit=50&offset=10",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
						UserID: 1,
						Limit:  50,
						Offset: 10,
					}).
					Return(&service.GetEmailsResult{
						Emails: []service.EmailResult{},
						Limit:  50,
						Offset: 10,
						Total:  0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:   "invalid query params fallback",
			userID: 1,
			query:  "?limit=abc&offset=-1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
						UserID: 1,
						Limit:  20,
						Offset: 0,
					}).
					Return(&service.GetEmailsResult{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "service error",
			userID: 1,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsByReceiver(gomock.Any(), gomock.Any()).
					Return(nil, service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "missing claims",
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid user id",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid query fallback",
			userID: 1,
			query:  "?limit=abc&offset=-1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
					UserID: 1,
					Limit:  20,
					Offset: 0,
				}).Return(&service.GetEmailsResult{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "custom pagination",
			userID: 1,
			query:  "?limit=50&offset=10",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
					UserID: 1,
					Limit:  50,
					Offset: 10,
				}).Return(&service.GetEmailsResult{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "service error",
			userID: 1,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Return(nil, service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "success",
			userID: 1,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsByReceiver(gomock.Any(), gomock.Any()).Return(&service.GetEmailsResult{}, nil)
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

			req := httptest.NewRequest(http.MethodGet, "/api/v1/emails"+tt.query, nil)

			if !tt.skipClaims {
				payload := &utils.JwtPayload{UserId: tt.userID}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.GetEmails(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp GetEmailsResponse
				json.NewDecoder(w.Body).Decode(&resp)
				if tt.expectedCount != 0 {
					require.Len(t, resp.Emails, tt.expectedCount)
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
		name           string
		userID         int64
		skipClaims     bool
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:   "success",
			userID: 1,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), gomock.Any()).
					Return(&service.GetMyEmailsResult{
						Emails: []service.MyEmailResult{
							{ID: 1, SenderID: 1, Header: "h", Body: "b", CreatedAt: now},
						},
						Limit:  20,
						Offset: 0,
						Total:  1,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "missing claims",
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsBySender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid user id",
			userID: 0,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailsBySender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: 1,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetEmailsBySender(gomock.Any(), gomock.Any()).
					Return(nil, service.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/myemails", nil)

			if !tt.skipClaims {
				payload := &utils.JwtPayload{UserId: tt.userID}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.GetMyEmails(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
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

	validRequest := SendEmailRequest{
		Header:    "h",
		Body:      "b",
		Receivers: []string{"a@smail.ru"},
	}

	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		customCtx      context.Context
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:        "missing claims",
			skipClaims:  true,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid claims type",
			customCtx:   context.WithValue(context.Background(), middleware.ClaimsKey, "wrong"),
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},

		{
			name:        "malformed json",
			userID:      1,
			requestBody: `{"header":`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty json object",
			userID:      1,
			requestBody: `{}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},

		{
			name:   "empty header",
			userID: 1,
			requestBody: SendEmailRequest{
				Body:      "b",
				Receivers: []string{"a@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "empty body",
			userID: 1,
			requestBody: SendEmailRequest{
				Header:    "h",
				Receivers: []string{"a@smail.ru"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "empty receivers",
			userID: 1,
			requestBody: SendEmailRequest{
				Header: "h",
				Body:   "b",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid receiver format",
			userID: 1,
			requestBody: SendEmailRequest{
				Header:    "h",
				Body:      "b",
				Receivers: []string{"bad"},
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},

		{
			name:        "service conflict",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrConflict)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:        "service bad request",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrBadRequest)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "service user not found",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "service email not found",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "service no valid receivers",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrNoValidReceivers)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "service access denied",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "service unknown error",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},

		{
			name:        "success",
			userID:      1,
			requestBody: validRequest,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Return(&service.SendEmailResult{
					ID:        1,
					SenderID:  1,
					Header:    "h",
					Body:      "b",
					CreatedAt: now,
				}, nil)
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

			h := &Handler{service: mockService}

			var body bytes.Buffer
			switch v := tt.requestBody.(type) {
			case string:
				body.WriteString(v)
			default:
				json.NewEncoder(&body).Encode(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/send", &body)

			if tt.customCtx != nil {
				req = req.WithContext(tt.customCtx)
			} else if !tt.skipClaims {
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, &utils.JwtPayload{
					UserId: tt.userID,
				})
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.SendEmail(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_SendEmail_NilBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().SendEmail(gomock.Any(), gomock.Any()).Times(0)

	h := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/send", nil)

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, &utils.JwtPayload{
		UserId: 1,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.SendEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_SendEmail_ResponseBody(t *testing.T) {
	now := time.Now()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)

	mockService.EXPECT().SendEmail(gomock.Any(), gomock.Any()).
		Return(&service.SendEmailResult{
			ID:        42,
			SenderID:  1,
			Header:    "hello",
			Body:      "world",
			CreatedAt: now,
		}, nil)

	h := &Handler{service: mockService}

	body, _ := json.Marshal(SendEmailRequest{
		Header:    "hello",
		Body:      "world",
		Receivers: []string{"a@smail.ru"},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/send", bytes.NewBuffer(body))

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, &utils.JwtPayload{
		UserId: 1,
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.SendEmail(w, req)

	var resp SendEmailResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.ID != 42 ||
		resp.SenderID != 1 ||
		resp.Header != "hello" ||
		resp.Body != "world" ||
		!resp.CreatedAt.Equal(now) {
		t.Errorf("response mismatch")
	}
}

func TestHandler_GetEmailByID(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		url            string
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "missing claims",
			skipClaims: true,
			url:        "/api/v1/emails/1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid user id",
			userID: 0,
			url:    "/api/v1/emails/1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid path short",
			userID: 1,
			url:    "/api/v1/emails",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid id format",
			userID: 1,
			url:    "/api/v1/emails/abc",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: 1,
			url:    "/api/v1/emails/1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Return(nil, service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "success",
			userID: 1,
			url:    "/api/v1/emails/1",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().GetEmailByID(gomock.Any(), gomock.Any()).Return(&service.GetEmailResult{
					ID: 1,
				}, nil)
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

			h := &Handler{service: mockService}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)

			if !tt.skipClaims {
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, &utils.JwtPayload{UserId: tt.userID})
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.GetEmailByID(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
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
		{
			name:        "missing claims",
			skipClaims:  true,
			requestBody: DeleteEmailRequest{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid json",
			userID:      1,
			requestBody: `{"email_id":`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid email id",
			userID: 1,
			requestBody: DeleteEmailRequest{
				EmailID: 0,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid user id",
			userID: 0,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForReceiver(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error not found",
			userID: 1,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any()).
					Return(service.ErrEmailNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "service access denied",
			userID: 1,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any()).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "service internal error",
			userID: 1,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any()).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "success",
			userID: 1,
			requestBody: DeleteEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForReceiver(gomock.Any(), gomock.Any()).
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
		{
			name:        "missing claims",
			skipClaims:  true,
			requestBody: DeleteMyEmailRequest{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid json",
			userID:      1,
			requestBody: `{"email_id":`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid email id",
			userID: 1,
			requestBody: DeleteMyEmailRequest{
				EmailID: 0,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid user id",
			userID: 0,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().DeleteEmailForSender(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: 1,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), gomock.Any()).
					Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "success",
			userID: 1,
			requestBody: DeleteMyEmailRequest{
				EmailID: 1,
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					DeleteEmailForSender(gomock.Any(), gomock.Any()).
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
		url            string
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
		{
			name:       "missing claims",
			skipClaims: true,
			url:        "/api/v1/emails/1/read",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid user id",
			userID: 0,
			url:    "/api/v1/emails/1/read",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid path",
			userID: 1,
			url:    "/api/v1/emails",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid id",
			userID: 1,
			url:    "/api/v1/emails/abc/read",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service error",
			userID: 1,
			url:    "/api/v1/emails/1/read",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Return(service.ErrAccessDenied)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "success",
			userID: 1,
			url:    "/api/v1/emails/1/read",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsRead(gomock.Any(), gomock.Any()).Return(nil)
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

			if tt.url == "" {
				tt.url = "/api/v1/emails/" + tt.emailID + "/read"
			}
			url := tt.url
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

func TestHandler_MarkEmailAsUnRead(t *testing.T) {
	tests := []struct {
		name           string
		emailID        string
		userID         int64
		skipClaims     bool
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name:       "successful mark as unread",
			emailID:    "1",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
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
					MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
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
					MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
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
					MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
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
				m.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid URL path - too short",
			emailID:    "",
			userID:     123,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing JWT claims",
			emailID:    "1",
			userID:     0,
			skipClaims: true,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID - zero",
			emailID:    "1",
			userID:     0,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID - negative",
			emailID:    "1",
			userID:     -5,
			skipClaims: false,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)
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
					MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
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

			url := "/api/v1/emails/" + tt.emailID + "/unread"
			req := httptest.NewRequest(http.MethodPut, url, nil)

			if tt.emailID != "" && tt.name != "invalid URL path - too short" {
				req = mux.SetURLVars(req, map[string]string{"id": tt.emailID})
			}
			if tt.name == "invalid URL path - too short" {
				req = httptest.NewRequest(http.MethodPut, "/api/v1/emails", nil)
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
			handler.MarkEmailAsUnRead(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_MarkEmailAsUnRead_InvalidClaimsType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	mockService.EXPECT().MarkEmailAsUnRead(gomock.Any(), gomock.Any()).Times(0)

	handler := &Handler{service: mockService}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/emails/1/unread", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	ctx := context.WithValue(req.Context(), middleware.ClaimsKey, "invalid claims type")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.MarkEmailAsUnRead(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestForwardEmailRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  ForwardEmailRequest
		want bool
	}{
		{"empty", ForwardEmailRequest{}, false},
		{"invalid email", ForwardEmailRequest{Receivers: []string{"bad"}}, false},
		{"valid", ForwardEmailRequest{Receivers: []string{"a@smail.ru"}}, true},
	}

	for _, tt := range tests {
		if tt.req.Validate() != tt.want {
			t.Fail()
		}
	}
}
