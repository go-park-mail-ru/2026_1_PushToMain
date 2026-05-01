package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/gorilla/mux"
)

type ChangeFolderRequest struct {
	Folder string `json:"folder"`
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	s, ok := mux.Vars(r)[key]
	if !ok || s == "" {
		return 0, errors.New("missing path param")
	}
	return strconv.ParseInt(s, 10, 64)
}

func parsePagination(r *http.Request) (int, int) {
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := 0
	if s := r.URL.Query().Get("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

// @Summary      Изменить расположение письма (spam/favorite/trash/folder-N)
// @Description  Помещает письмо в системную "папку" (флаг) или в кастомную папку (по name).
// @Tags         emails
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "ID письма"
// @Param        request body ChangeFolderRequest true "Папка"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/{id}/folder [put]
func (h *Handler) ChangeFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Change folder request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	var req ChangeFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}
	if req.Folder == "" {
		response.BadRequest(w)
		return
	}

	if err := h.service.ChangeFolder(r.Context(), service.ChangeFolderInput{
		UserID:  payload.UserId,
		EmailID: emailID,
		Folder:  req.Folder,
	}); err != nil {
		logger.Errorf("ChangeFolder failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary      Восстановить письмо из корзины
// @Tags         emails
// @Param        id   path      int  true  "ID письма"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/{id}/restore [put]
func (h *Handler) RestoreFromTrash(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Restore email request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	if err := h.service.RestoreFromTrash(r.Context(), service.ChangeFolderInput{
		UserID:  payload.UserId,
		EmailID: emailID,
	}); err != nil {
		logger.Errorf("RestoreFromTrash failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary      Получить письма из спама
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/spam [get]
func (h *Handler) GetSpamEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetSpamEmails(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetSpamEmails failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, r, result)
}

// @Summary      Получить письма из корзины
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/trash [get]
func (h *Handler) GetTrashEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetTrashEmails(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetTrashEmails failed: %v", err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, r, result)
}

func writeEmailsList(w http.ResponseWriter, r *http.Request, result *service.GetEmailsResult) {
	logger := middleware.GetLogger(r.Context())
	emails := make([]EmailResponse, len(result.Emails))
	for i, em := range result.Emails {
		emails[i] = EmailResponse{
			ID:            em.ID,
			SenderEmail:   em.SenderEmail,
			SenderName:    em.SenderName,
			SenderSurname: em.SenderSurname,
			ReceiverList:  em.ReceiverList,
			Header:        em.Header,
			Body:          em.Body,
			CreatedAt:     em.CreatedAt,
			IsRead:        em.IsRead,
		}
	}
	resp := GetEmailsResponse{
		Emails:      emails,
		Limit:       result.Limit,
		Offset:      result.Offset,
		Total:       result.Total,
		UnreadCount: result.UnreadCount,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
	}
}
