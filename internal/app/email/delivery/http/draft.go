package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type CreateDraftRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type UpdateDraftRequest struct {
	Header    string   `json:"header"`
	Body      string   `json:"body"`
	Receivers []string `json:"receivers"`
}

type DraftResponse struct {
	ID        int64     `json:"id"`
	SenderID  int64     `json:"sender_id"`
	Header    string    `json:"header"`
	Body      string    `json:"body"`
	Receivers []string  `json:"receivers"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetDraftsResponse struct {
	Drafts []DraftResponse `json:"drafts"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Total  int             `json:"total"`
}

func draftToResponse(r *service.DraftResult) DraftResponse {
	return DraftResponse{
		ID:        r.ID,
		SenderID:  r.SenderID,
		Header:    r.Header,
		Body:      r.Body,
		Receivers: r.Receivers,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// @Summary      Создать черновик
// @Description  Хотя бы одно из header/body/receivers должно быть непустым.
// @Tags         drafts
// @Accept       json
// @Produce      json
// @Param        request body CreateDraftRequest true "Черновик"
// @Success      201  {object}  DraftResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts [post]
func (h *Handler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Create draft request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	result, err := h.service.CreateDraft(r.Context(), service.CreateDraftInput{
		UserID:    payload.UserId,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("CreateDraft failed: %v", err)
		parseCommonErrors(err, w)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(draftToResponse(result)); err != nil {
		logger.Errorf("encode failed: %v", err)
	}
}

// @Summary      Обновить черновик (replace-семантика)
// @Tags         drafts
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "ID черновика"
// @Param        request body UpdateDraftRequest true "Черновик"
// @Success      200  {object}  DraftResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts/{id} [put]
func (h *Handler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Update draft request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	result, err := h.service.UpdateDraft(r.Context(), service.UpdateDraftInput{
		UserID:    payload.UserId,
		DraftID:   draftID,
		Header:    req.Header,
		Body:      req.Body,
		Receivers: req.Receivers,
	})
	if err != nil {
		logger.Errorf("UpdateDraft failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	if err := json.NewEncoder(w).Encode(draftToResponse(result)); err != nil {
		logger.Errorf("encode failed: %v", err)
	}
}

// @Summary      Получить черновик по ID
// @Tags         drafts
// @Produce      json
// @Param        id   path      int  true  "ID черновика"
// @Success      200  {object}  DraftResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts/{id} [get]
func (h *Handler) GetDraftByID(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	result, err := h.service.GetDraftByID(r.Context(), service.GetDraftInput{
		UserID: payload.UserId, DraftID: draftID,
	})
	if err != nil {
		logger.Errorf("GetDraftByID failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	if err := json.NewEncoder(w).Encode(draftToResponse(result)); err != nil {
		logger.Errorf("encode failed: %v", err)
	}
}

// @Summary      Получить список черновиков
// @Tags         drafts
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetDraftsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts [get]
func (h *Handler) GetDrafts(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetDrafts(r.Context(), service.GetDraftsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetDrafts failed: %v", err)
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("encode failed: %v", err)
	}
}

// @Summary      Удалить черновик безвозвратно
// @Tags         drafts
// @Param        id   path      int  true  "ID черновика"
// @Success      204
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts/{id} [delete]
func (h *Handler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	if err := h.service.DeleteDraft(r.Context(), service.DeleteDraftInput{
		UserID: payload.UserId, DraftID: draftID,
	}); err != nil {
		logger.Errorf("DeleteDraft failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Отправить черновик
// @Tags         drafts
// @Param        id   path      int  true  "ID черновика"
// @Success      200  {object}  SendEmailResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/drafts/{id}/send [post]
func (h *Handler) SendDraft(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	draftID, err := parsePathInt64(r, "id")
	if err != nil || draftID <= 0 {
		response.BadRequest(w)
		return
	}
	result, err := h.service.SendDraft(r.Context(), service.SendDraftInput{
		UserID: payload.UserId, DraftID: draftID,
	})
	if err != nil {
		logger.Errorf("SendDraft failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	resp := SendEmailResponse{
		ID:        result.ID,
		SenderID:  result.SenderID,
		Header:    result.Header,
		Body:      result.Body,
		CreatedAt: result.CreatedAt,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("encode failed: %v", err)
	}
}
