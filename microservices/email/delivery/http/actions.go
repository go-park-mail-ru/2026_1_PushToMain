package handler

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
)

func (h *Handler) batch(
	w http.ResponseWriter, r *http.Request, op string,
	fn func(ctx context.Context, in service.BatchInput) error,
) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	ids, ok := decodeIDs(w, r)
	if !ok {
		return
	}
	if err := fn(r.Context(), service.BatchInput{UserID: userID, EmailIDs: ids}); err != nil {
		logger.Errorf("%s: user_id=%d, ids=%v, err=%v", op, userID, ids, err)
		parseCommonErrors(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Trash(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Trash", h.service.Trash)
}

func (h *Handler) Untrash(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Untrash", h.service.Untrash)
}

func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Favorite", h.service.Favorite)
}

func (h *Handler) Unfavorite(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Unfavorite", h.service.Unfavorite)
}

func (h *Handler) Spam(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Spam", h.service.Spam)
}

func (h *Handler) Unspam(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Unspam", h.service.Unspam)
}

func (h *Handler) UnmarkSpamSenders(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "UnmarkSpamSenders", h.service.UnblockSenders)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "Delete", h.service.Delete)
}

func (h *Handler) BlockSenders(w http.ResponseWriter, r *http.Request) {
	h.batch(w, r, "BlockSenders", h.service.BlockSenders)
}
