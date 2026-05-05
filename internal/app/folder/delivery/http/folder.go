package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/folder/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/gorilla/mux"
)

type Service interface {
	CreateNewFolder(ctx context.Context, input service.CreateNewFolderInput) (*service.CreateNewFolderResult, error)
	ChangeFolderName(ctx context.Context, input service.ChangeFolderNameInput) error
	GetEmailsFromFolder(ctx context.Context, input service.GetEmailsFromFolderInput) (*service.GetEmailsFromFolderResult, error)
	AddEmailsInFolder(ctx context.Context, input service.AddEmailsInFolderInput) error
	DeleteEmailsFromFolder(ctx context.Context, input service.DeleteEmailsFromFolderInput) error
	DeleteFolder(ctx context.Context, input service.DeleteFolderInput) error
}

const MaxFolderNameLength = 255

var validFolderName = regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9\s\-_]+$`)

type CreateNewFolderRequest struct {
	FolderName string `json:"folder_name"`
}

type CreateNewFolderResponse struct {
	ID int64 `json:"folder_id"`
}

// @Summary      Создать новую папку
// @Description  Создаёт новую кастомную папку для писем
// @Tags         folders
// @Accept       json
// @Produce      json
// @Param        request body CreateNewFolderRequest true "Название папки"
// @Success      201  {object}  CreateNewFolderResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/new [post]
func (handler *Handler) CreateNewFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	var req CreateNewFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}
	if !req.Validate() {
		logger.Errorf("Validation failed, user_id=%d: invalid format", payload.UserId)
		response.BadRequest(w)
		return
	}
	result, err := handler.service.CreateNewFolder(r.Context(), service.CreateNewFolderInput{
		UserId:     payload.UserId,
		FolderName: req.FolderName,
	})
	if err != nil {
		logger.Errorf("Failed to create folder: %v", err)
		parseCommonErrors(err, w)
		return
	}
	resp := CreateNewFolderResponse{
		ID: result.ID,
	}

	logger.Debugf("Folder created successfully: user_id=%d, folder_id=%d",
		payload.UserId, result.ID)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

func (req *CreateNewFolderRequest) Validate() bool {

	if req.FolderName == "" || len(req.FolderName) > MaxFolderNameLength {
		return false
	}
	if !validFolderName.MatchString(req.FolderName) {
		return false
	}
	return true
}

type ChangeFolderNameRequest struct {
	FolderName string `json:"folder_name"`
}

// @Summary      Изменить название папки
// @Description  Изменяет название существующей кастомной папки
// @Tags         folders
// @Accept       json
// @Produce      json
// @Param        id      path      int                     true "ID папки"
// @Param        request body      ChangeFolderNameRequest true "Новое название папки"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      409  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/{id}/name [put]
func (handler *Handler) ChangeFolderName(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	vars := mux.Vars(r)
	folderIDStr := vars["folderID"]
	if folderIDStr == "" {
		logger.Errorf("Missing folder ID")
		response.BadRequest(w)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		logger.Errorf("Invalid folder ID: %s", folderIDStr)
		response.BadRequest(w)
		return
	}

	var req ChangeFolderNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}

	if !req.Validate() {
		logger.Errorf("Validation failed, user_id=%d: invalid format", payload.UserId)
		response.BadRequest(w)
		return
	}

	err = handler.service.ChangeFolderName(r.Context(), service.ChangeFolderNameInput{
		UserID:     payload.UserId,
		FolderID:   folderID,
		FolderName: req.FolderName,
	})
	if err != nil {
		logger.Errorf("Failed to change folder name: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Folder created successfully: user_id=%d",
		payload.UserId)
	w.WriteHeader(http.StatusOK)
}

func (req *ChangeFolderNameRequest) Validate() bool {

	if req.FolderName == "" || len(req.FolderName) > MaxFolderNameLength {
		return false
	}
	if !validFolderName.MatchString(req.FolderName) {
		return false
	}
	return true
}

type EmailResponse struct {
	ID            int64     `json:"id"`
	SenderEmail   string    `json:"sender_email"`
	SenderName    string    `json:"sender_name"`
	SenderSurname string    `json:"sender_surname"`
	ReceiverList  []string  `json:"receiver_list"`
	Header        string    `json:"header"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	IsRead        bool      `json:"is_read"`
	IsFavorite    bool      `json:"is_favorite"`
}

type GetEmailsFromFolderResponse struct {
	Emails      []EmailResponse `json:"emails"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
	Total       int             `json:"total"`
	UnreadCount int             `json:"unread_count"`
}

// @Summary      Получить письма из папки
// @Description  Возвращает список писем в указанной папке с пагинацией
// @Tags         folders
// @Produce      json
// @Param        id      path      int  true  "ID папки"
// @Param        limit   query     int  false "Количество записей на странице (default: 20, max: 100)"
// @Param        offset  query     int  false "Смещение для пагинации (default: 0)"
// @Success      200  {object}  GetEmailsFromFolderResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/{id} [get]
func (handler *Handler) GetEmailsFromFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	vars := mux.Vars(r)
	folderIDStr := vars["folderID"]
	if folderIDStr == "" {
		logger.Errorf("Missing folder ID")
		response.BadRequest(w)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		logger.Errorf("Invalid folder ID: %s", folderIDStr)
		response.BadRequest(w)
		return
	}

	limit, offset := GetLimitAndOffset(r)

	result, err := handler.service.GetEmailsFromFolder(r.Context(), service.GetEmailsFromFolderInput{
		UserID:   payload.UserId,
		FolderID: folderID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		logger.Errorf("Failed to get emails from folder: %v", err)
		parseCommonErrors(err, w)
		return
	}

	emails := make([]EmailResponse, len(result.Emails))
	for i, email := range result.Emails {
		emails[i] = EmailResponse{
			ID:            email.ID,
			SenderEmail:   email.SenderEmail,
			SenderName:    email.SenderName,
			SenderSurname: email.SenderSurname,
			ReceiverList:  email.ReceiverList,
			Header:        email.Header,
			Body:          email.Body,
			CreatedAt:     email.CreatedAt,
			IsRead:        email.IsRead,
			IsFavorite:    email.IsFavorite,
		}
	}

	resp := GetEmailsFromFolderResponse{
		Emails:      emails,
		Limit:       result.Limit,
		Offset:      result.Offset,
		Total:       result.Total,
		UnreadCount: result.UnreadCount,
	}

	logger.Debugf("Emails retrieved successfully: user_id=%d, count=%d, total=%d, unread=%d",
		payload.UserId, len(emails), result.Total, result.UnreadCount)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response: %v", err)
		response.InternalError(w)
		return
	}
}

type AddEmailsInFolderRequest struct {
	EmailsID []int64 `json:"emails_id"`
}

// @Summary      Добавить письма в папку
// @Description  Добавляет одно или несколько писем в указанную папку
// @Tags         folders
// @Accept       json
// @Produce      json
// @Param        id      path      int                       true "ID папки"
// @Param        request body      AddEmailsInFolderRequest  true "Список ID писем"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/{id}/add [post]
func (handler *Handler) AddEmailsInFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	vars := mux.Vars(r)
	folderIDStr := vars["folderID"]
	if folderIDStr == "" {
		logger.Errorf("Missing folder ID")
		response.BadRequest(w)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		logger.Errorf("Invalid folder ID: %s", folderIDStr)
		response.BadRequest(w)
		return
	}

	var req AddEmailsInFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}

	err = handler.service.AddEmailsInFolder(r.Context(), service.AddEmailsInFolderInput{
		UserID:   payload.UserId,
		FolderID: folderID,
		EmailsID: req.EmailsID,
	})
	if err != nil {
		logger.Errorf("Failed to add emails in folder: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("emails add successfully: user_id=%d",
		payload.UserId)
	w.WriteHeader(http.StatusOK)
}

type DeleteEmailsFromFolderRequest struct {
	EmailsID []int64 `json:"emails_id"`
}

// @Summary      Удалить письма из папки
// @Description  Удаляет указанные письма из папки
// @Tags         folders
// @Accept       json
// @Produce      json
// @Param        id      path      int                          true "ID папки"
// @Param        request body      DeleteEmailsFromFolderRequest true "Список ID писем"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/{id}/delete [delete]
func (handler *Handler) DeleteEmailsFromFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	vars := mux.Vars(r)
	folderIDStr := vars["folderID"]
	if folderIDStr == "" {
		logger.Errorf("Missing folder ID")
		response.BadRequest(w)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		logger.Errorf("Invalid folder ID: %s", folderIDStr)
		response.BadRequest(w)
		return
	}

	var req DeleteEmailsFromFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("Invalid request body, user_id=%d: %v", payload.UserId, err)
		response.BadRequest(w)
		return
	}

	err = handler.service.DeleteEmailsFromFolder(r.Context(), service.DeleteEmailsFromFolderInput{
		UserID:   payload.UserId,
		FolderID: folderID,
		EmailsID: req.EmailsID,
	})
	if err != nil {
		logger.Errorf("Failed to delete emails in folder: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("emails deleted successfully: user_id=%d",
		payload.UserId)
	w.WriteHeader(http.StatusOK)
}

// @Summary      Удалить папку
// @Description  Удаляет кастомную папку пользователя
// @Tags         folders
// @Produce      json
// @Param        id   path      int  true  "ID папки"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  response.ErrorResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Security     CookieAuth
// @Router       /api/v1/folder/{id} [delete]
func (handler *Handler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("Failed to get claims: %v", err)
		response.InternalError(w)
		return
	}

	vars := mux.Vars(r)
	folderIDStr := vars["folderID"]
	if folderIDStr == "" {
		logger.Errorf("Missing folder ID")
		response.BadRequest(w)
		return
	}

	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		logger.Errorf("Invalid folder ID: %s", folderIDStr)
		response.BadRequest(w)
		return
	}

	err = handler.service.DeleteFolder(r.Context(), service.DeleteFolderInput{
		UserID:   payload.UserId,
		FolderID: folderID,
	})
	if err != nil {
		logger.Errorf("Failed to delete folder: %v", err)
		parseCommonErrors(err, w)
		return
	}

	logger.Debugf("Folder deleted: user_id=%d, folder_id=%d", payload.UserId, folderID)

	w.WriteHeader(http.StatusOK)
}

func GetLimitAndOffset(r *http.Request) (limit, offset int) {
	limit = 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset = 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	return limit, offset
}
