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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// ====================== Batch handlers ======================

func TestHandler_Trash_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodPut, "/api/v1/emails/trash", IDsRequest{IDs: []int64{10, 20}})
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		Trash(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{10, 20}}).
		Return(nil)

	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Batch_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	req := requestWithContext(t, http.MethodPut, "/api/v1/emails/trash", IDsRequest{IDs: []int64{1}})
	// no claims injected
	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Batch_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/emails/trash", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Batch_ValidationEmptyIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodPut, "/api/v1/emails/trash", IDsRequest{IDs: []int64{}})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Batch_ValidationNonPositiveIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodPut, "/api/v1/emails/trash", IDsRequest{IDs: []int64{1, -1}})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Batch_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodPut, "/api/v1/emails/trash", IDsRequest{IDs: []int64{1}})
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		Trash(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	w := httptest.NewRecorder()
	h.Trash(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Quick smoke tests for all other batch endpoints to ensure they're wired correctly.
// They only test success to verify the runBatch wiring, because the logic is identical.
func TestBatchEndpoints_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		url     string
		setup   func()
	}{
		{
			name:    "Untrash",
			handler: h.Untrash,
			method:  http.MethodPut,
			url:     "/api/v1/emails/untrash",
			setup: func() {
				mockSvc.EXPECT().
					Untrash(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "Favorite",
			handler: h.Favorite,
			method:  http.MethodPut,
			url:     "/api/v1/emails/favorite",
			setup: func() {
				mockSvc.EXPECT().
					Favorite(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "Unfavorite",
			handler: h.Unfavorite,
			method:  http.MethodPut,
			url:     "/api/v1/emails/unfavorite",
			setup: func() {
				mockSvc.EXPECT().
					Unfavorite(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "Spam",
			handler: h.Spam,
			method:  http.MethodPut,
			url:     "/api/v1/emails/spam",
			setup: func() {
				mockSvc.EXPECT().
					Spam(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "Unspam",
			handler: h.Unspam,
			method:  http.MethodPut,
			url:     "/api/v1/emails/unspam",
			setup: func() {
				mockSvc.EXPECT().
					Unspam(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "UnmarkSpamSenders",
			handler: h.UnmarkSpamSenders,
			method:  http.MethodDelete,
			url:     "/api/v1/spam-senders",
			setup: func() {
				mockSvc.EXPECT().
					UnmarkSpamSenders(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
		{
			name:    "Delete",
			handler: h.Delete,
			method:  http.MethodDelete,
			url:     "/api/v1/emails",
			setup: func() {
				mockSvc.EXPECT().
					Delete(gomock.Any(), service.BatchInput{UserID: 1, EmailIDs: []int64{5}}).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			ctx := contextWithClaims(context.Background(), 1)
			req := requestWithContext(t, tt.method, tt.url, IDsRequest{IDs: []int64{5}})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			tt.handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ====================== Spam/Trash/Favorite listing handlers ======================

func TestHandler_GetSpamEmails_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/spam?limit=10&offset=5", nil)
	req = req.WithContext(ctx)

	expected := &service.GetEmailsResult{
		Emails: []service.EmailResult{
			{
				ID:            100,
				SenderEmail:   "spam@example.com",
				SenderName:    "Spam",
				SenderSurname: "Bot",
				ReceiverList:  []string{"me@example.com"},
				Header:        "Spam mail",
				Body:          "Buy something",
				CreatedAt:     time.Now(),
				IsRead:        false,
			},
		},
		Limit:       10,
		Offset:      5,
		Total:       1,
		UnreadCount: 1,
	}

	mockSvc.EXPECT().
		GetSpamEmails(gomock.Any(), service.GetEmailsInput{UserID: 1, Limit: 10, Offset: 5}).
		Return(expected, nil)

	w := httptest.NewRecorder()
	h.GetSpamEmails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp GetEmailsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Emails, 1)
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 5, resp.Offset)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, resp.UnreadCount)
	assert.Equal(t, "Spam mail", resp.Emails[0].Header)
}

func TestHandler_GetSpamEmails_DefaultPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/spam", nil)
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		GetSpamEmails(gomock.Any(), service.GetEmailsInput{UserID: 1, Limit: 20, Offset: 0}).
		Return(&service.GetEmailsResult{}, nil)

	w := httptest.NewRecorder()
	h.GetSpamEmails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetSpamEmails_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/spam", nil)

	w := httptest.NewRecorder()
	h.GetSpamEmails(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetSpamEmails_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/spam", nil)
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		GetSpamEmails(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	h.GetSpamEmails(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

// Similar tests for Trash and Favorite to ensure coverage.
func TestHandler_GetTrashEmails_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/trash", nil)
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		GetTrashEmails(gomock.Any(), service.GetEmailsInput{UserID: 1, Limit: 20, Offset: 0}).
		Return(&service.GetEmailsResult{}, nil)

	w := httptest.NewRecorder()
	h.GetTrashEmails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetFavoriteEmails_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	ctx := contextWithClaims(context.Background(), 1)
	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/favorite", nil)
	req = req.WithContext(ctx)

	mockSvc.EXPECT().
		GetFavoriteEmails(gomock.Any(), service.GetEmailsInput{UserID: 1, Limit: 20, Offset: 0}).
		Return(&service.GetEmailsResult{}, nil)

	w := httptest.NewRecorder()
	h.GetFavoriteEmails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test missing claims for the other listing endpoints
func TestHandler_GetTrashEmails_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/trash", nil)
	w := httptest.NewRecorder()
	h.GetTrashEmails(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetFavoriteEmails_MissingClaims(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	h := &Handler{service: mockSvc}

	req := requestWithContext(t, http.MethodGet, "/api/v1/emails/favorite", nil)
	w := httptest.NewRecorder()
	h.GetFavoriteEmails(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
