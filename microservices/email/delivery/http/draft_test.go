package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/email"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// CreateDraft
// ---------------------------------------------------------------------------

func TestHandler_CreateDraft(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	validReq := CreateDraftRequest{
		Header:    "Draft subject",
		Body:      "Draft body",
		Receivers: []string{"test@example.com"},
	}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts", validReq)
		req = req.WithContext(ctx)

		expectedResult := &service.DraftResult{
			ID:        100,
			SenderID:  1,
			Header:    validReq.Header,
			Body:      validReq.Body,
			Receivers: validReq.Receivers,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.EXPECT().
			CreateDraft(gomock.Any(), service.CreateDraftInput{
				UserID:    1,
				Header:    validReq.Header,
				Body:      validReq.Body,
				Receivers: validReq.Receivers,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.CreateDraft(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp DraftResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.ID, resp.ID)
		assert.Equal(t, expectedResult.Header, resp.Header)
		assert.ElementsMatch(t, expectedResult.Receivers, resp.Receivers)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts", validReq)
		w := httptest.NewRecorder()
		handler.CreateDraft(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drafts", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateDraft(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts", validReq)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			CreateDraft(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		w := httptest.NewRecorder()
		handler.CreateDraft(w, req)

		assert.NotEqual(t, http.StatusCreated, w.Code)
	})
}

// ---------------------------------------------------------------------------
// UpdateDraft
// ---------------------------------------------------------------------------

func TestHandler_UpdateDraft(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	validReq := UpdateDraftRequest{
		Header:    "Updated subject",
		Body:      "Updated body",
		Receivers: []string{"updated@example.com"},
	}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/api/v1/drafts/42", validReq)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"}) // inject path variable

		expectedResult := &service.DraftResult{
			ID:        42,
			SenderID:  1,
			Header:    validReq.Header,
			Body:      validReq.Body,
			Receivers: validReq.Receivers,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.EXPECT().
			UpdateDraft(gomock.Any(), service.UpdateDraftInput{
				UserID:    1,
				DraftID:   42,
				Header:    validReq.Header,
				Body:      validReq.Body,
				Receivers: validReq.Receivers,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.UpdateDraft(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp DraftResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), resp.ID)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodPut, "/api/v1/drafts/42", validReq)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})
		w := httptest.NewRecorder()
		handler.UpdateDraft(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid draft ID (non-numeric)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/api/v1/drafts/abc", validReq)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"}) // non-numeric
		w := httptest.NewRecorder()
		handler.UpdateDraft(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/drafts/42", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})
		w := httptest.NewRecorder()
		handler.UpdateDraft(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/api/v1/drafts/42", validReq)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})

		mockSvc.EXPECT().
			UpdateDraft(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("not found"))

		w := httptest.NewRecorder()
		handler.UpdateDraft(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetDraftByID
// ---------------------------------------------------------------------------

func TestHandler_GetDraftByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts/42", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})

		expectedResult := &service.DraftResult{
			ID:        42,
			SenderID:  1,
			Header:    "Hello",
			Body:      "Body",
			Receivers: []string{"recv@example.com"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockSvc.EXPECT().
			GetDraftByID(gomock.Any(), service.GetDraftInput{
				UserID:  1,
				DraftID: 42,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.GetDraftByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp DraftResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), resp.ID)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts/42", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})
		w := httptest.NewRecorder()
		handler.GetDraftByID(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid draft ID (non-numeric)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts/abc", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		w := httptest.NewRecorder()
		handler.GetDraftByID(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts/42", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})

		mockSvc.EXPECT().
			GetDraftByID(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("not found"))

		w := httptest.NewRecorder()
		handler.GetDraftByID(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetDrafts
// ---------------------------------------------------------------------------

func TestHandler_GetDrafts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success with default pagination", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts", nil)
		req = req.WithContext(ctx)

		expectedResult := &service.GetDraftsResult{
			Drafts: []service.DraftResult{
				{
					ID:        10,
					SenderID:  1,
					Header:    "Draft 1",
					Body:      "Body 1",
					Receivers: []string{"a@b.com"},
				},
			},
			Limit:  20,
			Offset: 0,
			Total:  1,
		}

		mockSvc.EXPECT().
			GetDrafts(gomock.Any(), service.GetDraftsInput{
				UserID: 1,
				Limit:  20,
				Offset: 0,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.GetDrafts(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp GetDraftsResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Drafts, 1)
		assert.Equal(t, 20, resp.Limit)
	})

	t.Run("custom pagination", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts?limit=10&offset=5", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			GetDrafts(gomock.Any(), service.GetDraftsInput{
				UserID: 1,
				Limit:  10,
				Offset: 5,
			}).
			Return(&service.GetDraftsResult{}, nil)

		w := httptest.NewRecorder()
		handler.GetDrafts(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts", nil)
		w := httptest.NewRecorder()
		handler.GetDrafts(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/drafts", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			GetDrafts(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		w := httptest.NewRecorder()
		handler.GetDrafts(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// DeleteDrafts
// ---------------------------------------------------------------------------

func TestHandler_DeleteDrafts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	// IDsRequest structure used by readIDsRequest
	type IDsRequest struct {
		IDs []int64 `json:"ids"`
	}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		reqBody := IDsRequest{IDs: []int64{1, 2, 3}}
		req := requestWithContext(t, http.MethodDelete, "/api/v1/drafts", reqBody)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			DeleteDrafts(gomock.Any(), service.DeleteDraftsInput{
				UserID:   1,
				DraftIDs: []int64{1, 2, 3},
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.DeleteDrafts(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodDelete, "/api/v1/drafts", IDsRequest{IDs: []int64{1}})
		w := httptest.NewRecorder()
		handler.DeleteDrafts(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid JSON body (handled by readIDsRequest)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/drafts", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.DeleteDrafts(w, req)

		// readIDsRequest will respond with BadRequest and return nil
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		reqBody := IDsRequest{IDs: []int64{4}}
		req := requestWithContext(t, http.MethodDelete, "/api/v1/drafts", reqBody)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			DeleteDrafts(gomock.Any(), gomock.Any()).
			Return(errors.New("not found"))

		w := httptest.NewRecorder()
		handler.DeleteDrafts(w, req)

		assert.NotEqual(t, http.StatusNoContent, w.Code)
	})
}

// ---------------------------------------------------------------------------
// SendDraft
// ---------------------------------------------------------------------------

func TestHandler_SendDraft(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts/42/send", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})

		expectedResult := &service.SendEmailResult{
			ID:        200,
			SenderID:  1,
			Header:    "Sent from draft",
			Body:      "Email body",
			CreatedAt: time.Now(),
		}

		mockSvc.EXPECT().
			SendDraft(gomock.Any(), service.SendDraftInput{
				UserID:  1,
				DraftID: 42,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.SendDraft(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp SendEmailResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.ID, resp.ID)
		assert.Equal(t, expectedResult.Header, resp.Header)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts/42/send", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})
		w := httptest.NewRecorder()
		handler.SendDraft(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid draft ID (non-numeric)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts/abc/send", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "abc"})
		w := httptest.NewRecorder()
		handler.SendDraft(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/api/v1/drafts/42/send", nil)
		req = req.WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"id": "42"})

		mockSvc.EXPECT().
			SendDraft(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("draft not found"))

		w := httptest.NewRecorder()
		handler.SendDraft(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}
