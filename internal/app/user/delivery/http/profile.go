package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type GetMeResponse struct {
    ID        int64  `json:"id"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    Surname   string `json:"surname"`
    ImagePath string `json:"image_path"`
}

func (handler *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
    logger := middleware.GetLogger(r.Context())

    claims, err := middleware.ClaimsFromContext(r.Context())
    if err != nil {
        logger.Errorf("failed to get claims from context: %v", err)
        response.InternalError(w)
        return
    }

    result, err := handler.service.GetMe(r.Context(), claims.UserId)
    if err != nil {
        logger.Errorf("failed to get user %d: %v", claims.UserId, err)
        response.InternalError(w)
        return
    }

    if err := json.NewEncoder(w).Encode(GetMeResponse{
        ID:        result.UserID,
        Email:     result.Email,
        Name:      result.Name,
        Surname:   result.Surname,
        ImagePath: result.ImagePath,
    }); err != nil {
        logger.Errorf("failed to encode response: %v", err)
        response.InternalError(w)
        return
    }
}

func (handler *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}
	if err = r.ParseMultipartForm(handler.cfg.MaxAvatarSize); err != nil {
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

	if !isValidImageType(header.Header.Get("Content-Type"), handler.cfg.AllowedTypes) {
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

func isValidImageType(contentType string, allowed []string) bool {
    for _, t := range allowed {
        if t == contentType {
            return true
        }
    }
    return false
}
