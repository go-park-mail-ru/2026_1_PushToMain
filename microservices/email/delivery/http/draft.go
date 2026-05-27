package handler

import (
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
)

func draftToResponse(r *service.DraftResult) DraftResponse {
	return DraftResponse{
		ID:        r.ID,
		SenderID:  r.SenderID,
		Header:    r.Header,
		Body:      r.Body,
		Receivers: r.Recipients,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (h *Handler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w)
		return
	}
	var req CreateDraftRequest
	if err := req.UnmarshalJSON(body); err != nil {
		response.BadRequest(w)
		return
	}
	result, err := h.service.CreateDraft(r.Context(), service.CreateDraftInput{
		UserID: userID, Header: req.Header, Body: req.Body, Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("CreateDraft: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	resp := DraftResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		Receivers: result.Recipients,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (h *Handler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w)
		return
	}
	var req UpdateDraftRequest
	if err := req.UnmarshalJSON(body); err != nil {
		response.BadRequest(w)
		return
	}
	result, err := h.service.UpdateDraft(r.Context(), service.UpdateDraftInput{
		UserID: userID, DraftID: draftID,
		Header: req.Header, Body: req.Body, Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("UpdateDraft: user_id=%d, draft_id=%d, err=%v", userID, draftID, err)
		parseCommonErrors(err, w)
		return
	}
	resp := DraftResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		Receivers: result.Recipients,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (h *Handler) GetDraftByID(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	result, err := h.service.GetDraftByID(r.Context(), service.GetDraftInput{
		UserID: userID, DraftID: draftID,
	})
	if err != nil {
		logger.Errorf("GetDraftByID: user_id=%d, draft_id=%d, err=%v", userID, draftID, err)
		parseCommonErrors(err, w)
		return
	}

	resp := DraftResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		Receivers: result.Recipients,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (h *Handler) GetDrafts(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetDrafts(r.Context(), service.GetDraftsInput{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetDrafts: user_id=%d, err=%v", userID, err)
		parseCommonErrors(err, w)
		return
	}
	out := make([]DraftResponse, len(result.Drafts))
	for i := range result.Drafts {
		out[i] = draftToResponse(&result.Drafts[i])
	}

	resp := GetDraftsResponse{
		Drafts: out, Limit: result.Limit, Offset: result.Offset, Total: result.Total,
	}

	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (h *Handler) DeleteDrafts(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	ids, ok := decodeIDs(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteDrafts(r.Context(), service.DeleteDraftsInput{
		UserID: userID, DraftIDs: ids,
	}); err != nil {
		logger.Errorf("DeleteDrafts: user_id=%d, ids=%v, err=%v", userID, ids, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SendDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	result, err := h.service.SendDraft(r.Context(), service.SendDraftInput{
		UserID: userID, DraftID: draftID,
	})
	if err != nil {
		logger.Errorf("SendDraft: user_id=%d, draft_id=%d, err=%v", userID, draftID, err)
		parseCommonErrors(err, w)
		return
	}

	resp := SendEmailResponse{
		ID: result.ID, SenderID: result.SenderID,
		Header: result.Header, Body: result.Body, CreatedAt: result.CreatedAt,
	}
	b, err := resp.MarshalJSON()
	if err != nil {
		response.InternalError(w)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}
