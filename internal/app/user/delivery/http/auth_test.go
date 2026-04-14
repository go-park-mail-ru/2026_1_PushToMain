package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"go.uber.org/mock/gomock"
)

func TestHandler_SignUp(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    SignUpRequest
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name: "successful signup",
			requestBody: SignUpRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@smail.ru",
				Password: "password123",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SignUp(gomock.Any(), service.SignUpInput{
						Email:    "john.doe@smail.ru",
						Password: "password123",
						Name:     "John",
						Surname:  "Doe",
					}).
					Return("test-token", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid email format",
			requestBody: SignUpRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "invalid@email.com",
				Password: "password123",
			},
			setupMock:      func(m *mocks.MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user already exists",
			requestBody: SignUpRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@smail.ru",
				Password: "password123",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SignUp(gomock.Any(), gomock.Any()).
					Return("", service.ErrUserAlreadyExists)
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

			handler := &Handler{
				service: mockService,
				cfg:     Config{TTL: time.Hour},
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Execute
			handler.SignUp(w, req)

			// Assert
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_SignIn(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    SignInRequest
		setupMock      func(*mocks.MockService)
		expectedStatus int
	}{
		{
			name: "successful signin",
			requestBody: SignInRequest{
				Email:    "john.doe@smail.ru",
				Password: "password123",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SignIn(gomock.Any(), service.SignInInput{
						Email:    "john.doe@smail.ru",
						Password: "password123",
					}).
					Return("test-token", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid email format",
			requestBody: SignInRequest{
				Email:    "invalid@email.com",
				Password: "password123",
			},
			setupMock:      func(m *mocks.MockService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user not found",
			requestBody: SignInRequest{
				Email:    "john.doe@smail.ru",
				Password: "password123",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SignIn(gomock.Any(), gomock.Any()).
					Return("", service.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "wrong password",
			requestBody: SignInRequest{
				Email:    "john.doe@smail.ru",
				Password: "wrong-password",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					SignIn(gomock.Any(), gomock.Any()).
					Return("", service.ErrWrongPassword)
			},
			expectedStatus: http.StatusUnauthorized,
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
				cfg:     Config{TTL: time.Hour},
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.SignIn(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "successful logout",
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"status": "ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
			w := httptest.NewRecorder()

			handler.Logout(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body["status"] != tt.expectedBody["status"] {
				t.Errorf("expected status %q, got %q", tt.expectedBody["status"], body["status"])
			}

			cookies := w.Result().Cookies()
			var sessionCookie *http.Cookie
			for _, c := range cookies {
				if c.Name == sessionTokenCookie {
					sessionCookie = c
					break
				}
			}
			if sessionCookie == nil {
				t.Error("session cookie not set")
				return
			}
			if sessionCookie.Value != "" {
				t.Errorf("expected empty cookie value, got %q", sessionCookie.Value)
			}
			if !sessionCookie.Expires.Before(time.Now()) {
				t.Error("expected cookie to be expired")
			}
			if !sessionCookie.HttpOnly {
				t.Error("expected HttpOnly flag to be true")
			}
			if sessionCookie.Path != "/" {
				t.Errorf("expected path '/', got %q", sessionCookie.Path)
			}
		})
	}
}

func TestHandler_GetCSRF(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedBody   *csrfResponse
	}{
		{
			name: "successful get csrf",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GenerateToken().
					Return("test-csrf-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &csrfResponse{
				Token: "test-csrf-token",
			},
		},
		{
			name: "service error",
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GenerateToken().
					Return("", errors.New("generation failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			tt.setupMock(mockService)

			handler := &Handler{service: mockService}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/csrf", nil)
			w := httptest.NewRecorder()

			handler.GetCSRF(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp csrfResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Token != tt.expectedBody.Token {
					t.Errorf("expected token %q, got %q", tt.expectedBody.Token, resp.Token)
				}

				cookies := w.Result().Cookies()
				var csrfCookie *http.Cookie
				for _, c := range cookies {
					if c.Name == "csrf_token" {
						csrfCookie = c
						break
					}
				}
				if csrfCookie == nil {
					t.Error("csrf cookie not set")
					return
				}
				if csrfCookie.Value != tt.expectedBody.Token {
					t.Errorf("expected cookie value %q, got %q", tt.expectedBody.Token, csrfCookie.Value)
				}
				if csrfCookie.HttpOnly {
					t.Error("expected HttpOnly to be false")
				}
				if csrfCookie.Path != "/" {
					t.Errorf("expected path '/', got %q", csrfCookie.Path)
				}
				if csrfCookie.SameSite != http.SameSiteLaxMode {
					t.Errorf("expected SameSite Lax, got %v", csrfCookie.SameSite)
				}
			}
		})
	}
}

func TestHandler_UpdatePassword(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		claimsValue    interface{}
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:       "successful password update",
			userID:     123,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdatePassword(gomock.Any(), service.UpdatePasswordInput{
						UserID:      123,
						OldPassword: "oldpass123",
						NewPassword: "newpass456",
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"status": "ok"},
		},
		{
			name:       "new password too short",
			userID:     123,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "short",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "incorrect old password",
			userID:     123,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "wrongold",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdatePassword(gomock.Any(), service.UpdatePasswordInput{
						UserID:      123,
						OldPassword: "wrongold",
						NewPassword: "newpass456",
					}).
					Return(service.ErrWrongPassword)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "user not found",
			userID:     123,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdatePassword(gomock.Any(), gomock.Any()).
					Return(service.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:       "service internal error",
			userID:     123,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdatePassword(gomock.Any(), gomock.Any()).
					Return(context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "missing JWT claims",
			userID:      0,
			skipClaims:  true,
			requestBody: UpdatePasswordRequest{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid claims type",
			userID:      123,
			skipClaims:  false,
			claimsValue: "invalid claims",
			requestBody: UpdatePasswordRequest{},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid user ID zero",
			userID:     0,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid user ID negative",
			userID:     -5,
			skipClaims: false,
			requestBody: UpdatePasswordRequest{
				OldPassword: "oldpass123",
				NewPassword: "newpass456",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "malformed request body",
			userID:      123,
			skipClaims:  false,
			requestBody: `{"old_password": 123}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty request body",
			userID:      123,
			skipClaims:  false,
			requestBody: nil,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any()).Times(0)
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

			req := httptest.NewRequest(http.MethodPut, "/api/v1/user/password", &body)
			req.Header.Set("Content-Type", "application/json")

			if !tt.skipClaims {
				var claims interface{}
				if tt.claimsValue != nil {
					claims = tt.claimsValue
				} else {
					claims = &utils.JwtPayload{
						UserId: tt.userID,
						Exp:    time.Now().Add(time.Hour).Unix(),
					}
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, claims)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.UpdatePassword(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != nil {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp["status"] != tt.expectedBody["status"] {
					t.Errorf("expected status %q, got %q", tt.expectedBody["status"], resp["status"])
				}
			}
		})
	}
}
