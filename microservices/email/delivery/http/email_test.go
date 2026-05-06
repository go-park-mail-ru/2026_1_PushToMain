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

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	mocks "github.com/go-park-mail-ru/2026_1_PushToMain/mocks/app/email"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// contextWithClaims injects a valid JWT payload using the same function
// that AuthMiddleware does.
func contextWithClaims(ctx context.Context, userID int64) context.Context {
	claims := &utils.JwtPayload{UserId: userID}
	return middleware.ContextWithClaims(ctx, claims)
}

// requestWithContext creates an HTTP request with an optional JSON body.
// It does not manipulate the logger – your GetLogger fallback handles that.
func requestWithContext(t *testing.T, method, url string, body interface{}) *http.Request {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// ---------------------------------------------------------------------------
// SendEmail
// ---------------------------------------------------------------------------

func TestHandler_SendEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	validReq := SendEmailRequest{
		Header:    "Test subject",
		Body:      "Test body",
		Receivers: []string{"alice@example.com"},
	}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/send", validReq)
		req = req.WithContext(ctx)

		expectedResult := &service.SendEmailResult{
			ID:        100,
			SenderID:  1,
			Header:    validReq.Header,
			Body:      validReq.Body,
			CreatedAt: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		}
		mockSvc.EXPECT().
			SendEmail(gomock.Any(), service.SendEmailInput{
				UserId:    1,
				Header:    validReq.Header,
				Body:      validReq.Body,
				Receivers: validReq.Receivers,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.SendEmail(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp SendEmailResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.ID, resp.ID)
		assert.Equal(t, expectedResult.SenderID, resp.SenderID)
		assert.Equal(t, expectedResult.Header, resp.Header)
		assert.Equal(t, expectedResult.Body, resp.Body)
		assert.True(t, expectedResult.CreatedAt.Equal(resp.CreatedAt))
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodPost, "/send", validReq)
		w := httptest.NewRecorder()
		handler.SendEmail(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := httptest.NewRequest(http.MethodPost, "/send", bytes.NewBufferString("{bad json"))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.SendEmail(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation fails (empty receivers)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/send", SendEmailRequest{
			Header:    "subject",
			Body:      "body",
			Receivers: []string{},
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.SendEmail(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/send", validReq)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			SendEmail(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("some error"))

		w := httptest.NewRecorder()
		handler.SendEmail(w, req)

		// parseCommonErrors will decide the status; we just verify it's not 200
		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// ForwardEmail
// ---------------------------------------------------------------------------

func TestHandler_ForwardEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	validReq := ForwardEmailRequest{
		EmailID:   42,
		Receivers: []string{"bob@example.com"},
	}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/forward", validReq)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			ForwardEmail(gomock.Any(), service.ForwardEmailInput{
				UserID:    1,
				EmailID:   validReq.EmailID,
				Receivers: validReq.Receivers,
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.ForwardEmail(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid email_id <= 0", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPost, "/forward", ForwardEmailRequest{
			EmailID:   0,
			Receivers: []string{"bob@example.com"},
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ForwardEmail(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing claims", func(t *testing.T) {
		req := requestWithContext(t, http.MethodPost, "/forward", validReq)
		w := httptest.NewRecorder()
		handler.ForwardEmail(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetEmails
// ---------------------------------------------------------------------------

func TestHandler_GetEmails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success with default pagination", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/emails", nil)
		req = req.WithContext(ctx)

		serviceResult := &service.GetEmailsResult{
			Emails: []service.EmailResult{
				{
					ID:            10,
					SenderEmail:   "alice@example.com",
					SenderName:    "Alice",
					SenderSurname: "Smith",
					ReceiverList:  []string{"me@example.com"},
					Header:        "Hello",
					Body:          "Body",
					CreatedAt:     time.Now(),
					IsRead:        false,
				},
			},
			Limit:       20,
			Offset:      0,
			Total:       1,
			UnreadCount: 1,
		}

		mockSvc.EXPECT().
			GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
				UserID: 1,
				Limit:  20,
				Offset: 0,
			}).
			Return(serviceResult, nil)

		w := httptest.NewRecorder()
		handler.GetEmails(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp GetEmailsResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Emails, 1)
		assert.Equal(t, 20, resp.Limit)
		assert.Equal(t, 0, resp.Offset)
		assert.Equal(t, 1, resp.Total)
		assert.Equal(t, 1, resp.UnreadCount)
	})

	t.Run("custom pagination", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/emails?limit=10&offset=5", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			GetEmailsByReceiver(gomock.Any(), service.GetEmailsInput{
				UserID: 1,
				Limit:  10,
				Offset: 5,
			}).
			Return(&service.GetEmailsResult{}, nil)

		w := httptest.NewRecorder()
		handler.GetEmails(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid user ID", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 0) // invalid
		req := requestWithContext(t, http.MethodGet, "/emails", nil)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetEmails(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ---------------------------------------------------------------------------
// GetMyEmails
// ---------------------------------------------------------------------------

func TestHandler_GetMyEmails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/myemails", nil)
		req = req.WithContext(ctx)

		serviceResult := &service.GetMyEmailsResult{
			Emails: []service.MyEmailResult{
				{
					ID:              5,
					SenderID:        1,
					Header:          "Sent",
					Body:            "Body",
					CreatedAt:       time.Now(),
					IsRead:          true,
					ReceiversEmails: []string{"other@example.com"},
				},
			},
			Limit:  20,
			Offset: 0,
			Total:  1,
		}

		mockSvc.EXPECT().
			GetEmailsBySender(gomock.Any(), service.GetMyEmailsInput{
				UserID: 1,
				Limit:  20,
				Offset: 0,
			}).
			Return(serviceResult, nil)

		w := httptest.NewRecorder()
		handler.GetMyEmails(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp GetMyEmailsResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Emails, 1)
	})
}

// ---------------------------------------------------------------------------
// GetEmailByID
// ---------------------------------------------------------------------------

func TestHandler_GetEmailByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		// Use the full API path as the handler expects.
		req := requestWithContext(t, http.MethodGet, "/api/v1/emails/42", nil)
		req = req.WithContext(ctx)

		expectedResult := &service.GetEmailResult{
			ID:              42,
			SenderID:        2,
			SenderEmail:     "bob@example.com",
			SenderName:      "Bob",
			SenderSurname:   "Jones",
			Header:          "Important",
			Body:            "Please reply",
			CreatedAt:       time.Now(),
			SenderImagePath: "/avatars/bob.jpg",
			ReceiverList:    []string{"me@example.com"},
		}

		mockSvc.EXPECT().
			GetEmailByID(gomock.Any(), service.GetEmailInput{
				UserID:  1,
				EmailID: 42,
			}).
			Return(expectedResult, nil)

		w := httptest.NewRecorder()
		handler.GetEmailByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp GetEmailResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult.ID, resp.ID)
		assert.Equal(t, expectedResult.SenderImagePath, resp.SenderImagePath)
	})

	t.Run("invalid path (too short)", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/emails", nil)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetEmailByID(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("emailID not numeric", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/emails/abc", nil)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetEmailByID(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service returns error", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodGet, "/api/v1/emails/99", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			GetEmailByID(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("not found"))

		w := httptest.NewRecorder()
		handler.GetEmailByID(w, req)

		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// MarkEmailAsRead / MarkEmailAsUnRead (single)
// ---------------------------------------------------------------------------

func TestHandler_MarkEmailAsRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		// Full path as required by the route.
		req := requestWithContext(t, http.MethodPut, "/api/v1/emails/10/read", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
				UserID:  1,
				EmailID: []int64{10},
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.MarkEmailAsRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid path -> bad request", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/emails/read", nil)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.MarkEmailAsRead(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_MarkEmailAsUnRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/api/v1/emails/10/unread", nil)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
				UserID:  1,
				EmailID: []int64{10},
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.MarkEmailAsUnRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ---------------------------------------------------------------------------
// MarkEmailsAsRead / MarkEmailsAsUnRead (batch)
// ---------------------------------------------------------------------------

func TestHandler_MarkEmailsAsRead_Batch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		reqBody := MarkEmailsAsReadRequest{EmailIDs: []int64{1, 2, 3}}
		req := requestWithContext(t, http.MethodPut, "/emails/read", reqBody)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			MarkEmailAsRead(gomock.Any(), service.MarkAsReadInput{
				UserID:  1,
				EmailID: []int64{1, 2, 3},
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.MarkEmailsAsRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("empty email_ids -> bad request", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		req := requestWithContext(t, http.MethodPut, "/emails/read", MarkEmailsAsReadRequest{
			EmailIDs: []int64{},
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.MarkEmailsAsRead(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandler_MarkEmailsAsUnRead_Batch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockService(ctrl)
	handler := &Handler{service: mockSvc}

	t.Run("success", func(t *testing.T) {
		ctx := contextWithClaims(context.Background(), 1)
		reqBody := MarkEmailsAsReadRequest{EmailIDs: []int64{5, 6}}
		req := requestWithContext(t, http.MethodPut, "/emails/unread", reqBody)
		req = req.WithContext(ctx)

		mockSvc.EXPECT().
			MarkEmailAsUnRead(gomock.Any(), service.MarkAsReadInput{
				UserID:  1,
				EmailID: []int64{5, 6},
			}).
			Return(nil)

		w := httptest.NewRecorder()
		handler.MarkEmailsAsUnRead(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
