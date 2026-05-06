package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testHandler struct {
	*Handler
	mockService *mocks.MockService
}

func setupTest(t *testing.T) (*gomock.Controller, *testHandler) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockService(ctrl)
	cfg := Config{
		TTL:           24 * time.Hour,
		MaxAvatarSize: 10 << 20,
		AllowedTypes:  []string{"image/jpeg", "image/png"},
	}
	h := New(mockSvc, cfg)
	return ctrl, &testHandler{Handler: h, mockService: mockSvc}
}

func newRequestWithClaims(t *testing.T, method, url string, body interface{}, payload *utils.JwtPayload) *http.Request {
	t.Helper()
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if payload != nil {
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
	}
	return req
}

func createJPEGMultipartBody(t *testing.T, fieldName, fileName string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	h.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	data := make([]byte, 512)
	data[0] = 0xFF
	data[1] = 0xD8
	data[2] = 0xFF
	data[3] = 0xE0
	_, err = part.Write(data)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	return body, writer.FormDataContentType()
}

// ------- GetMe -------
func TestHandler_GetMe(t *testing.T) {
	payload := &utils.JwtPayload{UserId: 42, Exp: time.Now().Add(time.Hour).Unix()}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		birth := time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC)
		male := true

		th.mockService.EXPECT().
			GetMe(gomock.Any(), payload.UserId).
			Return(&service.GetMeResult{
				UserID:    42,
				Email:     "john@smail.ru",
				Name:      "John",
				Surname:   "Doe",
				ImagePath: "/avatars/42.jpg",
				IsMale:    &male,
				Birthdate: &birth,
				Folders: []service.Folder{
					{ID: 1, Name: "inbox"},
					{ID: 2, Name: "sent"},
				},
			}, nil)

		req := newRequestWithClaims(t, http.MethodGet, "/me", nil, payload)
		rec := httptest.NewRecorder()

		th.GetMe(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp GetMeResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, int64(42), resp.ID)
		assert.Equal(t, "john@smail.ru", resp.Email)
		assert.Equal(t, "John", resp.Name)
		assert.Equal(t, "Doe", resp.Surname)
		assert.Equal(t, "/avatars/42.jpg", resp.ImagePath)
		assert.Equal(t, &male, resp.IsMale)
		assert.Equal(t, &birth, resp.Birthdate)
		assert.Len(t, resp.Folders, 2)
		assert.Equal(t, int64(1), resp.Folders[0].ID)
		assert.Equal(t, "inbox", resp.Folders[0].Name)
		assert.Equal(t, int64(2), resp.Folders[1].ID)
		assert.Equal(t, "sent", resp.Folders[1].Name)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(t, http.MethodGet, "/me", nil, nil)
		rec := httptest.NewRecorder()

		th.GetMe(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GetMe(gomock.Any(), payload.UserId).
			Return(nil, service.ErrUserNotFound)

		req := newRequestWithClaims(t, http.MethodGet, "/me", nil, payload)
		rec := httptest.NewRecorder()

		th.GetMe(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GetMe(gomock.Any(), payload.UserId).
			Return(nil, errors.New("db error"))

		req := newRequestWithClaims(t, http.MethodGet, "/me", nil, payload)
		rec := httptest.NewRecorder()

		th.GetMe(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("empty folders", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GetMe(gomock.Any(), payload.UserId).
			Return(&service.GetMeResult{
				UserID:  42,
				Email:   "john@smail.ru",
				Folders: []service.Folder{},
			}, nil)

		req := newRequestWithClaims(t, http.MethodGet, "/me", nil, payload)
		rec := httptest.NewRecorder()

		th.GetMe(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp GetMeResponse
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Empty(t, resp.Folders)
	})
}

// ------- UploadAvatar -------
func TestHandler_UploadAvatar(t *testing.T) {
	payload := &utils.JwtPayload{UserId: 42, Exp: time.Now().Add(time.Hour).Unix()}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UploadAvatar(gomock.Any(), gomock.AssignableToTypeOf(service.UploadAvatarInput{})).
			Return("/avatars/42.jpg", nil)

		body, contentType := createJPEGMultipartBody(t, "avatar", "test.jpg")
		req := httptest.NewRequest(http.MethodPost, "/avatar", body)
		req.Header.Set("Content-Type", contentType)
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UploadAvatar(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]string
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "/avatars/42.jpg", resp["image_path"])
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		body, contentType := createJPEGMultipartBody(t, "avatar", "test.jpg")
		req := httptest.NewRequest(http.MethodPost, "/avatar", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()

		th.UploadAvatar(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("not multipart", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPost, "/avatar", bytes.NewReader([]byte("not multipart")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UploadAvatar(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing avatar field", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		body, contentType := createJPEGMultipartBody(t, "wrongfield", "test.jpg")
		req := httptest.NewRequest(http.MethodPost, "/avatar", body)
		req.Header.Set("Content-Type", contentType)
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UploadAvatar(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UploadAvatar(gomock.Any(), gomock.AssignableToTypeOf(service.UploadAvatarInput{})).
			Return("", errors.New("s3 error"))

		body, contentType := createJPEGMultipartBody(t, "avatar", "test.jpg")
		req := httptest.NewRequest(http.MethodPost, "/avatar", body)
		req.Header.Set("Content-Type", contentType)
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UploadAvatar(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ------- UpdateProfile -------
func TestHandler_UpdateProfile(t *testing.T) {
	payload := &utils.JwtPayload{UserId: 42, Exp: time.Now().Add(time.Hour).Unix()}
	birth := time.Date(2000, 2, 20, 0, 0, 0, 0, time.UTC)
	male := true

	validBody := UpdateProfileRequest{
		Name:      "John",
		Surname:   "Doe",
		Birthdate: &birth,
		IsMale:    &male,
	}

	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UpdateProfile(gomock.Any(), service.UpdateProfileInput{
				UserID:    payload.UserId,
				Name:      validBody.Name,
				Surname:   validBody.Surname,
				IsMale:    validBody.IsMale,
				Birthdate: validBody.Birthdate,
			}).
			Return(nil)

		req := newRequestWithClaims(t, http.MethodPut, "/profile", validBody, payload)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp map[string]string
		err := json.NewDecoder(rec.Body).Decode(&resp)
		require.NoError(t, err)
		assert.Equal(t, "profile updated successfully", resp["status"])
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(t, http.MethodPut, "/profile", validBody, nil)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("invalid user id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		invalidPayload := &utils.JwtPayload{UserId: 0, Exp: time.Now().Add(time.Hour).Unix()}
		req := newRequestWithClaims(t, http.MethodPut, "/profile", validBody, invalidPayload)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPut, "/profile", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("empty name", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		body := UpdateProfileRequest{Name: "", Surname: "Doe"}
		req := newRequestWithClaims(t, http.MethodPut, "/profile", body, payload)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("empty surname", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		body := UpdateProfileRequest{Name: "John", Surname: ""}
		req := newRequestWithClaims(t, http.MethodPut, "/profile", body, payload)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			UpdateProfile(gomock.Any(), gomock.Any()).
			Return(service.ErrUserNotFound)

		req := newRequestWithClaims(t, http.MethodPut, "/profile", validBody, payload)
		rec := httptest.NewRecorder()

		th.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
