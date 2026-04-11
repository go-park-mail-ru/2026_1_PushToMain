package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

var (
	Kilobyte int64 = 1024
	Megabyte int64 = 1024 * Kilobyte
)

func (handler *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}
	if err = r.ParseMultipartForm(1 * Megabyte); err != nil {
		logger.Errorf("failed to parse multipart form: %v", err)
		response.BadRequest(w)
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		logger.Errorf("failed to get avatar file: %v", err)
		response.BadRequest(w)
		return
	}
	defer file.Close()

	if !isValidImageType(header.Header.Get("Content-Type")) {
		logger.Infof("invalid image type: %s", header.Header.Get("Content-Type"))
		response.BadRequest(w)
		return
	}
	imagePath, err := handler.service.UploadAvatar(r.Context(), service.UploadAvatarInput{
		File:   file,
		Size:   header.Size,
		UserID: claims.UserId,
	})
	if err != nil {
		logger.Errorf("failed to upload avatar for user %d: %v", claims.UserId, err)
		response.InternalError(w)
		return
	}

	logger.Infof("avatar uploaded for user %d: %s", claims.UserId, imagePath)

	responseBody := map[string]string{
		"image_path": imagePath,
	}

	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		logger.Errorf("failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func isValidImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg":
		return true
	}
	return false
}
