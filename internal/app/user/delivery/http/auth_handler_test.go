package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
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
			expectedStatus: http.StatusUnauthorized,
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
