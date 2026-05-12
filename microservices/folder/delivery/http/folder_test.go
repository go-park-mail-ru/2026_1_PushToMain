package handler

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
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/folder"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// ---------- test helpers ----------

type testHandler struct {
	*Handler
	mockService *mocks.MockService
}

func setupTest(t *testing.T) (*gomock.Controller, *testHandler) {
	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockService(ctrl)
	return ctrl, &testHandler{
		Handler:     New(mockSvc),
		mockService: mockSvc,
	}
}

func newRequestWithClaims(method, url string, body any, payload *utils.JwtPayload) *http.Request {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewReader(b))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	if payload != nil {
		ctx := middleware.ContextWithClaims(req.Context(), payload)
		req = req.WithContext(ctx)
	}
	return req
}

func validClaims() *utils.JwtPayload {
	return &utils.JwtPayload{UserId: 42, Exp: time.Now().Add(time.Hour).Unix()}
}

// ---------- CreateNewFolderRequest.Validate ----------

func TestCreateNewFolderRequest_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input CreateNewFolderRequest
		valid bool
	}{
		{"empty name", CreateNewFolderRequest{""}, false},
		{"too long", CreateNewFolderRequest{string(make([]byte, 256))}, false},
		{"invalid chars", CreateNewFolderRequest{"bad!!"}, false},
		{"valid", CreateNewFolderRequest{"Work"}, true},
		{"valid with spaces", CreateNewFolderRequest{"My Folder"}, true},
		{"valid with dash", CreateNewFolderRequest{"spam-2024"}, true},
		{"valid with underscore", CreateNewFolderRequest{"read_later"}, true},
		{"russian letters", CreateNewFolderRequest{"Работа"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.input.Validate())
		})
	}
}

// ---------- ChangeFolderNameRequest.Validate ----------

func TestChangeFolderNameRequest_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input ChangeFolderNameRequest
		valid bool
	}{
		{"empty name", ChangeFolderNameRequest{""}, false},
		{"too long", ChangeFolderNameRequest{string(make([]byte, 300))}, false},
		{"invalid chars", ChangeFolderNameRequest{"!!!"}, false},
		{"valid", ChangeFolderNameRequest{"Archive"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.input.Validate())
		})
	}
}

// ---------- GetLimitAndOffset ----------

func TestGetLimitAndOffset(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		limit, offset := GetLimitAndOffset(r)
		assert.Equal(t, 20, limit)
		assert.Equal(t, 0, offset)
	})
	t.Run("custom valid limit and offset", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?limit=50&offset=10", nil)
		limit, offset := GetLimitAndOffset(r)
		assert.Equal(t, 50, limit)
		assert.Equal(t, 10, offset)
	})
	t.Run("limit exceeds max", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?limit=200", nil)
		limit, _ := GetLimitAndOffset(r)
		assert.Equal(t, 20, limit)
	})
	t.Run("negative limit", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?limit=-5", nil)
		limit, _ := GetLimitAndOffset(r)
		assert.Equal(t, 20, limit)
	})
	t.Run("negative offset", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?offset=-5", nil)
		_, offset := GetLimitAndOffset(r)
		assert.Equal(t, 0, offset)
	})
	t.Run("non-numeric", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?limit=abc&offset=xyz", nil)
		limit, offset := GetLimitAndOffset(r)
		assert.Equal(t, 20, limit)
		assert.Equal(t, 0, offset)
	})
}

// ---------- CreateNewFolder ----------

func TestHandler_CreateNewFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			CreateNewFolder(gomock.Any(), service.CreateNewFolderInput{UserId: 42, FolderName: "Work"}).
			Return(&service.CreateNewFolderResult{ID: 10}, nil)

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: "Work"}, validClaims())
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code) // 200 in source, though docs say 201
		var resp CreateNewFolderResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, int64(10), resp.ID)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: "Work"}, nil)
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("invalid json", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPost, "/folder/new",
			bytes.NewReader([]byte("bad json")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), validClaims())
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation fails", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: ""}, validClaims())
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – folder already exists", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			CreateNewFolder(gomock.Any(), gomock.Any()).
			Return(nil, service.ErrFolderAlreadyExists)

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: "Work"}, validClaims())
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("service error – max folders reached", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			CreateNewFolder(gomock.Any(), gomock.Any()).
			Return(nil, service.ErrMaxFoldersReached)

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: "Work"}, validClaims())
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("service error – internal", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			CreateNewFolder(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db down"))

		req := newRequestWithClaims(http.MethodPost, "/folder/new",
			CreateNewFolderRequest{FolderName: "Work"}, validClaims())
		rec := httptest.NewRecorder()

		th.CreateNewFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// ---------- ChangeFolderName ----------

func TestHandler_ChangeFolderName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			ChangeFolderName(gomock.Any(), service.ChangeFolderNameInput{
				UserID: 42, FolderID: 5, FolderName: "Updated",
			}).
			Return(nil)

		req := newRequestWithClaims(http.MethodPut, "/folder/5/name",
			ChangeFolderNameRequest{FolderName: "Updated"}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPut, "/folder/5/name",
			ChangeFolderNameRequest{FolderName: "Updated"}, nil)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("missing folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPut, "/folder//name",
			ChangeFolderNameRequest{FolderName: "Updated"}, validClaims())
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPut, "/folder/abc/name",
			ChangeFolderNameRequest{FolderName: "Updated"}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "abc"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPut, "/folder/5/name",
			bytes.NewReader([]byte("bad")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), validClaims())
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation fails", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPut, "/folder/5/name",
			ChangeFolderNameRequest{FolderName: ""}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – not found", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			ChangeFolderName(gomock.Any(), gomock.Any()).
			Return(service.ErrFolderNotFound)

		req := newRequestWithClaims(http.MethodPut, "/folder/5/name",
			ChangeFolderNameRequest{FolderName: "Valid"}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("service error – access denied", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			ChangeFolderName(gomock.Any(), gomock.Any()).
			Return(service.ErrAccessDenied)

		req := newRequestWithClaims(http.MethodPut, "/folder/5/name",
			ChangeFolderNameRequest{FolderName: "Valid"}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.ChangeFolderName(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------- GetEmailsFromFolder ----------

func TestHandler_GetEmailsFromFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		now := time.Now()
		th.mockService.EXPECT().
			GetEmailsFromFolder(gomock.Any(), service.GetEmailsFromFolderInput{
				UserID: 42, FolderID: 5, Limit: 20, Offset: 0,
			}).
			Return(&service.GetEmailsFromFolderResult{
				Emails: []service.EmailFromFolderResult{
					{ID: 100, SenderEmail: "a@smail.ru", SenderName: "A",
						SenderSurname: "B", Header: "H", Body: "B",
						CreatedAt: now, IsRead: false},
				},
				Limit: 20, Offset: 0, Total: 1, UnreadCount: 1,
			}, nil)

		req := newRequestWithClaims(http.MethodGet, "/folder/5", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.GetEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp GetEmailsFromFolderResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Len(t, resp.Emails, 1)
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodGet, "/folder/5", nil, nil)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.GetEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("missing folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodGet, "/folder/", nil, validClaims())
		rec := httptest.NewRecorder()

		th.GetEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodGet, "/folder/abc", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "abc"})
		rec := httptest.NewRecorder()

		th.GetEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – not found", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			GetEmailsFromFolder(gomock.Any(), gomock.Any()).
			Return(nil, service.ErrFolderNotFound)

		req := newRequestWithClaims(http.MethodGet, "/folder/5", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.GetEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------- AddEmailsInFolder ----------

func TestHandler_AddEmailsInFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			AddEmailsInFolder(gomock.Any(), service.AddEmailsInFolderInput{
				UserID: 42, FolderID: 5, EmailsID: []int64{100, 200},
			}).
			Return(nil)

		req := newRequestWithClaims(http.MethodPost, "/folder/5/add",
			AddEmailsInFolderRequest{EmailsID: []int64{100, 200}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPost, "/folder/5/add",
			AddEmailsInFolderRequest{EmailsID: []int64{100}}, nil)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("missing folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPost, "/folder//add",
			AddEmailsInFolderRequest{EmailsID: []int64{100}}, validClaims())
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodPost, "/folder/abc/add",
			AddEmailsInFolderRequest{EmailsID: []int64{100}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "abc"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodPost, "/folder/5/add",
			bytes.NewReader([]byte("bad")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), validClaims())
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – access denied", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			AddEmailsInFolder(gomock.Any(), gomock.Any()).
			Return(service.ErrAccessDenied)

		req := newRequestWithClaims(http.MethodPost, "/folder/5/add",
			AddEmailsInFolderRequest{EmailsID: []int64{100}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("service error – empty emails list", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			AddEmailsInFolder(gomock.Any(), gomock.Any()).
			Return(service.ErrEmptyEmailsList)

		req := newRequestWithClaims(http.MethodPost, "/folder/5/add",
			AddEmailsInFolderRequest{EmailsID: []int64{}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.AddEmailsInFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// ---------- DeleteEmailsFromFolder ----------

func TestHandler_DeleteEmailsFromFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			DeleteEmailsFromFolder(gomock.Any(), service.DeleteEmailsFromFolderInput{
				UserID: 42, FolderID: 5, EmailsID: []int64{100},
			}).
			Return(nil)

		req := newRequestWithClaims(http.MethodDelete, "/folder/5/delete",
			DeleteEmailsFromFolderRequest{EmailsID: []int64{100}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder/5/delete",
			DeleteEmailsFromFolderRequest{EmailsID: []int64{100}}, nil)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("missing folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder//delete",
			DeleteEmailsFromFolderRequest{EmailsID: []int64{100}}, validClaims())
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder/abc/delete",
			DeleteEmailsFromFolderRequest{EmailsID: []int64{100}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "abc"})
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := httptest.NewRequest(http.MethodDelete, "/folder/5/delete",
			bytes.NewReader([]byte("bad")))
		req.Header.Set("Content-Type", "application/json")
		ctx := middleware.ContextWithClaims(req.Context(), validClaims())
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – access denied", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			DeleteEmailsFromFolder(gomock.Any(), gomock.Any()).
			Return(service.ErrAccessDenied)

		req := newRequestWithClaims(http.MethodDelete, "/folder/5/delete",
			DeleteEmailsFromFolderRequest{EmailsID: []int64{100}}, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteEmailsFromFolder(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------- DeleteFolder ----------

func TestHandler_DeleteFolder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			DeleteFolder(gomock.Any(), service.DeleteFolderInput{
				UserID: 42, FolderID: 5,
			}).
			Return(nil)

		req := newRequestWithClaims(http.MethodDelete, "/folder/5", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("no claims", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder/5", nil, nil)
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("missing folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder/", nil, validClaims())
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid folder id", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		req := newRequestWithClaims(http.MethodDelete, "/folder/abc", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "abc"})
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("service error – not found", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			DeleteFolder(gomock.Any(), gomock.Any()).
			Return(service.ErrFolderNotFound)

		req := newRequestWithClaims(http.MethodDelete, "/folder/5", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("service error – access denied", func(t *testing.T) {
		ctrl, th := setupTest(t)
		defer ctrl.Finish()

		th.mockService.EXPECT().
			DeleteFolder(gomock.Any(), gomock.Any()).
			Return(service.ErrAccessDenied)

		req := newRequestWithClaims(http.MethodDelete, "/folder/5", nil, validClaims())
		req = mux.SetURLVars(req, map[string]string{"folderID": "5"})
		rec := httptest.NewRecorder()

		th.DeleteFolder(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
