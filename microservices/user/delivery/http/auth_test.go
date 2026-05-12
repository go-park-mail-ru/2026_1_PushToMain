package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// SignUp
func TestHandler_SignUp(t *testing.T) {
	validBody := SignUpRequest{
		Name:     "John",
		Surname:  "Doe",
		Email:    "john@smail.ru",
		Password: "password123",
	}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignUp(gomock.Any(), service.SignUpInput{
				Email:    validBody.Email,
				Password: validBody.Password,
				Name:     validBody.Name,
				Surname:  validBody.Surname,
			}).
			Return("test-token", nil)

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignUp(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, sessionTokenCookie, cookies[0].Name)
		assert.Equal(t, "test-token", cookies[0].Value)

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("invalid json", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignUp(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation failed", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		invalidBody := SignUpRequest{Email: "bad"}
		bodyBytes, _ := json.Marshal(invalidBody)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignUp(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("user already exists", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignUp(gomock.Any(), gomock.Any()).
			Return("", service.ErrUserAlreadyExists)

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignUp(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignUp(gomock.Any(), gomock.Any()).
			Return("", errors.New("db error"))

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignUp(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// SignIn
func TestHandler_SignIn(t *testing.T) {
	validBody := SignInRequest{
		Email:    "john@smail.ru",
		Password: "password123",
	}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignIn(gomock.Any(), service.SignInInput{
				Email:    validBody.Email,
				Password: validBody.Password,
			}).
			Return("test-token", nil)

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignIn(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, sessionTokenCookie, cookies[0].Name)
		assert.Equal(t, "test-token", cookies[0].Value)

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("invalid json", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignIn(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation failed", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		invalidBody := SignInRequest{Email: "bad"}
		bodyBytes, _ := json.Marshal(invalidBody)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignIn(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignIn(gomock.Any(), gomock.Any()).
			Return("", service.ErrUserNotFound)

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignIn(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("wrong password", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			SignIn(gomock.Any(), gomock.Any()).
			Return("", service.ErrWrongPassword)

		bodyBytes, _ := json.Marshal(validBody)
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		th.SignIn(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// Logout
func TestHandler_Logout(t *testing.T) {
	ctrl, th := setupTest(t)
	defer ctrl.Finish()

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	th.Logout(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, sessionTokenCookie, cookies[0].Name)
	assert.Empty(t, cookies[0].Value)
	assert.True(t, cookies[0].Expires.Before(time.Now()))

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "ok", resp["status"])
}

// GetCSRF
func TestHandler_GetCSRF(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GenerateToken().
			Return("csrf-test-token", nil)

		req := httptest.NewRequest(http.MethodGet, "/csrf", nil)
		rec := httptest.NewRecorder()

		th.GetCSRF(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		cookies := rec.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "csrf_token" {
				csrfCookie = c
				break
			}
		}
		require.NotNil(t, csrfCookie)
		assert.Equal(t, "csrf-test-token", csrfCookie.Value)

		var resp csrfResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "csrf-test-token", resp.Token)
	})

	t.Run("token generation fails", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GenerateToken().
			Return("", errors.New("crypto error"))

		req := httptest.NewRequest(http.MethodGet, "/csrf", nil)
		rec := httptest.NewRecorder()

		th.GetCSRF(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// UpdatePassword
func TestHandler_UpdatePassword(t *testing.T) {
	payload := &utils.JwtPayload{UserId: 42, Exp: time.Now().Add(time.Hour).Unix()}
	validBody := UpdatePasswordRequest{
		OldPassword: "oldpass",
		NewPassword: "newpass123",
	}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UpdatePassword(gomock.Any(), service.UpdatePasswordInput{
				UserID:      payload.UserId,
				OldPassword: validBody.OldPassword,
				NewPassword: validBody.NewPassword,
			}).
			Return(nil)

		req := newRequestWithClaims(t, http.MethodPut, "/password", validBody, payload)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(t, http.MethodPut, "/password", validBody, nil)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("invalid user id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		invalidPayload := &utils.JwtPayload{UserId: 0, Exp: time.Now().Add(time.Hour).Unix()}
		req := newRequestWithClaims(t, http.MethodPut, "/password", validBody, invalidPayload)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPut, "/password", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("short new password", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		body := UpdatePasswordRequest{OldPassword: "old", NewPassword: "short"}
		req := newRequestWithClaims(t, http.MethodPut, "/password", body, payload)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("wrong old password", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UpdatePassword(gomock.Any(), gomock.Any()).
			Return(service.ErrWrongPassword)

		req := newRequestWithClaims(t, http.MethodPut, "/password", validBody, payload)
		rec := httptest.NewRecorder()

		th.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
