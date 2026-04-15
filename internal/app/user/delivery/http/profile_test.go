package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/mocks"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_GetMe(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedBody   *GetMeResponse
	}{
		{
			name:   "success",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetMe(gomock.Any(), int64(123)).
					Return(&service.GetMeResult{
						UserID:    123,
						Email:     "user@smail.ru",
						Name:      "John",
						Surname:   "Doe",
						ImagePath: "/avatars/123.jpg",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &GetMeResponse{
				ID:        123,
				Email:     "user@smail.ru",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatars/123.jpg",
			},
		},
		{
			name:   "user not found",
			userID: 999,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetMe(gomock.Any(), int64(999)).
					Return(nil, service.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal service error",
			userID: 123,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					GetMe(gomock.Any(), int64(123)).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "missing claims",
			skipClaims:     true,
			setupMock:      func(m *mocks.MockService) {},
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

			req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			if !tt.skipClaims {
				payload := &utils.JwtPayload{
					UserId: tt.userID,
					Exp:    time.Now().Add(time.Hour).Unix(),
				}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.GetMe(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var resp GetMeResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, &resp)
			}
		})
	}
}

func TestHandler_UploadAvatar(t *testing.T) {
	cfg := &Config{
		MaxAvatarSize: 10 << 20,
		AllowedTypes:  []string{"image/jpeg", "image/png"},
	}

	validJPEGHeader := []byte{0xFF, 0xD8, 0xFF}

	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		setupRequest   func() (*http.Request, error)
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:   "success",
			userID: 123,
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("avatar", "test.jpg")
				if err != nil {
					return nil, err
				}
				// Write valid JPEG header plus some dummy data
				part.Write(validJPEGHeader)
				part.Write([]byte("some image data"))
				writer.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UploadAvatar(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, input service.UploadAvatarInput) (string, error) {
						assert.Equal(t, int64(123), input.UserID)
						assert.NotNil(t, input.File)
						// Size is len(validJPEGHeader) + len("some image data")
						assert.Equal(t, int64(len(validJPEGHeader)+len("some image data")), input.Size)
						return "/avatars/123.jpg", nil
					})
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"image_path": "/avatars/123.jpg"},
		},
		{
			name:   "service upload error",
			userID: 123,
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("avatar", "test.jpg")
				part.Write(validJPEGHeader)
				writer.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UploadAvatar(gomock.Any(), gomock.Any()).
					Return("", service.ErrUploadAvatar)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "invalid file type",
			userID: 123,
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("avatar", "test.txt")
				part.Write([]byte("text content"))
				writer.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UploadAvatar(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "missing avatar file",
			userID: 123,
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UploadAvatar(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "missing claims",
			skipClaims: true,
			setupRequest: func() (*http.Request, error) {
				return httptest.NewRequest(http.MethodPost, "/api/v1/avatar", nil), nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UploadAvatar(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "exceeds max size",
			userID: 123,
			setupRequest: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, _ := writer.CreateFormFile("avatar", "large.jpg")
				// Write enough data to exceed limit (ParseMultipartForm will fail)
				part.Write(make([]byte, cfg.MaxAvatarSize+1))
				writer.Close()
				req := httptest.NewRequest(http.MethodPost, "/api/v1/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UploadAvatar(gomock.Any(), gomock.Any()).Times(0)
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

			handler := &Handler{service: mockService, cfg: *cfg}

			req, err := tt.setupRequest()
			require.NoError(t, err)

			if !tt.skipClaims {
				payload := &utils.JwtPayload{UserId: tt.userID, Exp: time.Now().Add(time.Hour).Unix()}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.UploadAvatar(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var resp map[string]string
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}
		})
	}
}

func TestHandler_UpdateProfile(t *testing.T) {
	tests := []struct {
		name           string
		userID         int64
		skipClaims     bool
		requestBody    interface{}
		setupMock      func(*mocks.MockService)
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:   "success",
			userID: 123,
			requestBody: UpdateProfileRequest{
				Name:    "John",
				Surname: "Doe",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdateProfile(gomock.Any(), service.UpdateProfileInput{
						UserID:  123,
						Name:    "John",
						Surname: "Doe",
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"status": "profile updated successfully"},
		},
		{
			name:   "update only name",
			userID: 123,
			requestBody: UpdateProfileRequest{
				Name:    "Jane",
				Surname: "",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdateProfile(gomock.Any(), service.UpdateProfileInput{
						UserID:  123,
						Name:    "Jane",
						Surname: "",
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   map[string]string{"status": "profile updated successfully"},
		},
		{
			name:   "update only surname",
			userID: 123,
			requestBody: UpdateProfileRequest{
				Name:    "",
				Surname: "Smith",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdateProfile(gomock.Any(), service.UpdateProfileInput{
						UserID:  123,
						Name:    "",
						Surname: "Smith",
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "empty name and surname",
			userID: 123,
			requestBody: UpdateProfileRequest{
				Name:    "",
				Surname: "",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "invalid request body",
			userID:      123,
			requestBody: `{"name": 123}`,
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: 999,
			requestBody: UpdateProfileRequest{
				Name:    "John",
				Surname: "Doe",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdateProfile(gomock.Any(), service.UpdateProfileInput{
						UserID:  999,
						Name:    "John",
						Surname: "Doe",
					}).
					Return(service.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "service internal error",
			userID: 123,
			requestBody: UpdateProfileRequest{
				Name:    "John",
				Surname: "Doe",
			},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().
					UpdateProfile(gomock.Any(), gomock.Any()).
					Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "missing claims",
			skipClaims:     true,
			requestBody:    UpdateProfileRequest{Name: "John", Surname: "Doe"},
			setupMock:      func(m *mocks.MockService) {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "invalid user ID zero",
			userID:      0,
			requestBody: UpdateProfileRequest{Name: "John", Surname: "Doe"},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Times(0)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "invalid user ID negative",
			userID:      -1,
			requestBody: UpdateProfileRequest{Name: "John", Surname: "Doe"},
			setupMock: func(m *mocks.MockService) {
				m.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Times(0)
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

			handler := &Handler{service: mockService}

			var body io.Reader
			if tt.requestBody != nil {
				switch v := tt.requestBody.(type) {
				case string:
					body = strings.NewReader(v)
				default:
					b, _ := json.Marshal(v)
					body = bytes.NewReader(b)
				}
			}

			req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", body)
			req.Header.Set("Content-Type", "application/json")

			if !tt.skipClaims {
				payload := &utils.JwtPayload{UserId: tt.userID, Exp: time.Now().Add(time.Hour).Unix()}
				ctx := context.WithValue(req.Context(), middleware.ClaimsKey, payload)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.UpdateProfile(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var resp map[string]string
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}
		})
	}
}
