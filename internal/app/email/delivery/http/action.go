package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/gorilla/mux"
)

func parsePathInt64(r *http.Request, key string) (int64, error) {
	s, ok := mux.Vars(r)[key]
	if !ok || s == "" {
		return 0, errors.New("missing path param")
	}
	return strconv.ParseInt(s, 10, 64)
}

type IDsRequest struct {
	IDs []int64 `json:"ids"`
}

func (r IDsRequest) Validate() error {
	if len(r.IDs) == 0 {
		return errors.New("ids is required and must be non-empty")
	}
	for _, id := range r.IDs {
		if id <= 0 {
			return errors.New("ids must be positive")
		}
	}
	return nil
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

// readIDsRequest — общий парсинг + валидация для всех массовых ручек.
// Возвращает nil, если пакет невалидный (ответ уже отправлен внутри).
func readIDsRequest(w http.ResponseWriter, r *http.Request) *IDsRequest {
	logger := middleware.GetLogger(r.Context())
	var req IDsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return nil
	}
	if err := req.Validate(); err != nil {
		logger.Warnf("Validation failed: %v", err)
		response.BadRequest(w)
		return nil
	}
	return &req
}

// runBatch — общий шаблон: получить claims → распарсить ids → вызвать сервис → ответить.
// fn делает только обращение к сервису, всё остальное — здесь.
func runBatch(w http.ResponseWriter, r *http.Request, opName string,
	fn func(ctx context.Context, in service.BatchInput) error) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("%s request received", opName)

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("%s: failed to get claims: %v", opName, err)
		response.InternalError(w)
		return
	}

	req := readIDsRequest(w, r)
	if req == nil {
		return
	}

	if err := fn(r.Context(), service.BatchInput{UserID: payload.UserId, EmailIDs: req.IDs}); err != nil {
		logger.Errorf("%s failed: user_id=%d, ids=%v, err=%v", opName, payload.UserId, req.IDs, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary      Переместить письма в корзину
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/trash [put]
func (h *Handler) Trash(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Trash", h.service.Trash)
}

// @Summary      Вернуть письма из корзины
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/untrash [put]
func (h *Handler) Untrash(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Untrash", h.service.Untrash)
}

// @Summary      Добавить письма в избранное
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/favorite [put]
func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Favorite", h.service.Favorite)
}

// @Summary      Убрать письма из избранного
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/unfavorite [put]
func (h *Handler) Unfavorite(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Unfavorite", h.service.Unfavorite)
}

// @Summary      Пометить письма как спам
// @Description  Помечает письма is_spam=true и добавляет отправителей в spam_senders.
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/spam [put]
func (h *Handler) Spam(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Spam", h.service.Spam)
}

// @Summary      Снять метку спам с писем
// @Description  Снимает is_spam с указанных писем (только там, где текущий юзер — получатель)
// @Description  и удаляет соответствующих отправителей из персонального списка spam_senders.
// @Description  На остальные письма этих же отправителей, ранее помеченные спамом, не влияет —
// @Description  они остаются в спаме до явного вызова unspam с их id.
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/unspam [put]
func (h *Handler) Unspam(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Unspam", h.service.Unspam)
}

// @Summary      Удалить письма (двухэтапно)
// @Description  Если письмо не в корзине — переместить в корзину; если уже в корзине — удалить физически.
// @Description  Решение принимается отдельно для каждого письма из массива.
// @Tags         emails
// @Accept       json
// @Param        request body IDsRequest true "Список ID писем"
// @Success      200  "Success"
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	runBatch(w, r, "Delete", h.service.Delete)
}

// @Summary      Получить письма из спама
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Кол-во записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/spam [get]
func (h *Handler) GetSpamEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Get spam emails request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("GetSpamEmails: failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetSpamEmails(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetSpamEmails failed: user_id=%d, err=%v", payload.UserId, err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, r, result)
}

// @Summary      Получить письма из корзины
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Кол-во записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/trash [get]
func (h *Handler) GetTrashEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Get trash emails request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("GetTrashEmails: failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetTrashEmails(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetTrashEmails failed: user_id=%d, err=%v", payload.UserId, err)
		parseCommonErrors(err, w)
		return
	}
	writeEmailsList(w, r, result)
}

// @Summary      Получить избранные письма
// @Tags         emails
// @Produce      json
// @Param        limit   query     int  false  "Кол-во записей (default 20, max 100)"
// @Param        offset  query     int  false  "Смещение (default 0)"
// @Success      200  {object}  GetEmailsResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/emails/favorite [get]
func (h *Handler) GetFavoriteEmails(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Get favorite emails request received")
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("GetFavoriteEmails: failed to get claims: %v", err)
		response.InternalError(w)
		return
	}
	limit, offset := parsePagination(r)
	result, err := h.service.GetFavoriteEmails(r.Context(), service.GetEmailsInput{
		UserID: payload.UserId, Limit: limit, Offset: offset,
	})
	if err != nil {
		logger.Errorf("GetFavoriteEmails failed: user_id=%d, err=%v", payload.UserId, err)
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
			IsFavorite:    em.IsStarred,
		}
	}
	resp := GetEmailsResponse{
		Emails: emails, Limit: result.Limit, Offset: result.Offset,
		Total: result.Total, UnreadCount: result.UnreadCount,
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("writeEmailsList: encode failed: %v", err)
		response.InternalError(w)
	}
}
