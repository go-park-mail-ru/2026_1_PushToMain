package http

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"
)

type Folder struct {
	ID   int64  `json:"folder_id"`
	Name string `json:"folder_name"`
}
type GetMeResponse struct {
	ID        int64    `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Surname   string   `json:"surname"`
	ImagePath string   `json:"image_path"`
	Folders   []Folder `json:"folder"`

	IsMale    *bool      `json:"is_male,omitempty"`
	Birthdate *time.Time `json:"birthdate,omitempty"`
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
	if errors.Is(err, service.ErrUserNotFound) {
		logger.Errorf("user (id: %d) not found: %v", claims.UserId, err)
		response.NotFound(w)
		return
	} else if err != nil {
		logger.Errorf("failed to get user %d: %v", claims.UserId, err)
		response.InternalError(w)
		return
	}

	folders := make([]Folder, len(result.Folders))
	for i, f := range result.Folders {
		folders[i] = Folder{
			ID:   f.ID,
			Name: f.Name,
		}
	}

	if err := json.NewEncoder(w).Encode(GetMeResponse{
		ID:        result.UserID,
		Email:     result.Email,
		Name:      result.Name,
		Surname:   result.Surname,
		ImagePath: result.ImagePath,
		IsMale:    result.IsMale,
		Birthdate: result.Birthdate,
		Folders:   folders,
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

	if !isValidFileType(file, handler.cfg.AllowedTypes) {
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

type UpdateProfileRequest struct {
	Name      string     `json:"name"`
	Surname   string     `json:"surname"`
	Birthdate *time.Time `json:"birthdate"` // ISO-8601 формат 2000-02-20
	IsMale    *bool      `json:"is_male"`
}

// @Summary      Обновить профиль пользователя
// @Description  Обновляет имя и фамилию авторизованного пользователя
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body UpdateProfileRequest true "Новые имя и фамилия"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/profile [put]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	logger.Infof("Update profile request received")

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	if payload.UserId <= 0 {
		logger.Warnf("Invalid user ID: %d", payload.UserId)
		response.Unauthorized(w)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warnf("Invalid request body: %v", err)
		response.BadRequest(w)
		return
	}

	if req.Name == "" || req.Surname == "" {
		logger.Warnf("Name and surname are empty")
		response.BadRequest(w)
		return
	}

	err = h.service.UpdateProfile(r.Context(), service.UpdateProfileInput{
		UserID:    payload.UserId,
		Name:      req.Name,
		Surname:   req.Surname,
		IsMale:    req.IsMale,
		Birthdate: req.Birthdate,
	})
	if err != nil {
		logger.Errorf("Failed to update profile: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Infof("Profile updated successfully, user_id=%d", payload.UserId)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "profile updated successfully",
	})
}

func isValidFileType(file multipart.File, allowed []string) bool {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	_, _ = file.Seek(0, io.SeekStart)

	contentType := http.DetectContentType(buf[:n])
	return slices.Contains(allowed, contentType)
}
