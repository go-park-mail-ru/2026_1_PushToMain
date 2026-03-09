package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/service"
)

type mockAuthService struct {
	signUpFunc func(ctx context.Context, input service.SignUpInput) (string, error)
	signInFunc func(ctx context.Context, input service.SignInInput) (string, error)
}

func (m *mockAuthService) SignUp(ctx context.Context, input service.SignUpInput) (string, error) {
	return m.signUpFunc(ctx, input)
}

func (m *mockAuthService) SignIn(ctx context.Context, input service.SignInInput) (string, error) {
	return m.signInFunc(ctx, input)
}

func TestAuthHandler_SignUp(t *testing.T) {

	tests := []struct {
		name           string
		body           interface{}
		mockResponse   string
		mockError      error
		expectedStatus int
	}{
		{
			name: "success",
			body: SignUpRequest{
				Name:     "Ivan",
				Surname:  "Ivanov",
				Email:    "test@test.com",
				Password: "123456",
			},
			mockResponse:   "token123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bad json",
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user already exists",
			body: SignUpRequest{
				Name:     "Ivan",
				Surname:  "Ivanov",
				Email:    "test@test.com",
				Password: "123456",
			},
			mockError:      service.ErrUserAlreadyExists,
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			mockService := &mockAuthService{
				signUpFunc: func(ctx context.Context, input service.SignUpInput) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := NewAuthHandler(mockService)

			bodyBytes, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(bodyBytes))
			rec := httptest.NewRecorder()

			handler.SignUp(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_SignIn(t *testing.T) {

	tests := []struct {
		name           string
		body           SignInRequest
		mockResponse   string
		mockError      error
		expectedStatus int
	}{
		{
			name: "success",
			body: SignInRequest{
				Email:    "test@test.com",
				Password: "123456",
			},
			mockResponse:   "token123",
			expectedStatus: http.StatusOK,
		},
		{
			name: "wrong password",
			body: SignInRequest{
				Email:    "test@test.com",
				Password: "wrong",
			},
			mockError:      service.ErrWrongPassword,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			body: SignInRequest{
				Email:    "unknown@test.com",
				Password: "123",
			},
			mockError:      service.ErrUserNotFound,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			mockService := &mockAuthService{
				signInFunc: func(ctx context.Context, input service.SignInInput) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			handler := NewAuthHandler(mockService)

			bodyBytes, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(bodyBytes))
			rec := httptest.NewRecorder()

			handler.SignIn(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
