package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/gorilla/mux"
)

const (
	maxAttachmentSize = 25 << 20 // 25 MB
	formFileKey       = "file"
)

// GetAttachments godoc
// @Summary      List attachments of an email
// @Tags         attachments
// @Produce      json
// @Param        id  path      int  true  "Email ID"
// @Success      200 {object}  GetAttachmentsResponse
// @Router       /api/v1/email/emails/{id}/attachments [get]
func (h *Handler) GetAttachments(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	result, err := h.service.GetAttachments(r.Context(), service.GetAttachmentsInput{
		UserID:  userID,
		EmailID: emailID,
	})
	if err != nil {
		logger.Errorf("GetAttachments: user_id=%d, email_id=%d, err=%v", userID, emailID, err)
		parseCommonErrors(err, w)
		return
	}

	out := make([]AttachmentResponse, len(result.Attachments))
	for i, a := range result.Attachments {
		out[i] = attachmentToResponse(a)
	}
	resp := GetAttachmentsResponse{Attachments: out}

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

// UploadAttachment godoc
// @Summary      Upload an attachment to an email
// @Tags         attachments
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      int   true  "Email ID"
// @Param        file  formData  file  true  "File to upload"
// @Success      201   {object}  AttachmentResponse
// @Router       /api/v1/email/emails/{id}/attachments [post]
func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	if err := r.ParseMultipartForm(maxAttachmentSize); err != nil {
		response.BadRequest(w)
		return
	}

	file, header, err := r.FormFile(formFileKey)
	if err != nil {
		response.BadRequest(w)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Errorf("file close error: %v", err)
		}
	}()

	if header.Size > maxAttachmentSize {
		response.BadRequest(w)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := h.service.UploadAttachment(r.Context(), service.UploadAttachmentInput{
		UserID:      userID,
		EmailID:     emailID,
		Filename:    header.Filename,
		ContentType: contentType,
		Size:        header.Size,
		File:        file,
	})
	if err != nil {
		logger.Errorf("UploadAttachment: user_id=%d, email_id=%d, err=%v", userID, emailID, err)
		parseCommonErrors(err, w)
		return
	}

	resp := AttachmentResponse{
		ID:          result.ID,
		EmailID:     result.EmailID,
		FileName:    result.FileName,
		ContentType: result.ContentType,
		SizeBytes:   result.SizeBytes,
		CreatedAt:   result.CreatedAt,
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

// DownloadAttachment godoc
// @Summary      Download an attachment
// @Tags         attachments
// @Produce      application/octet-stream
// @Param        id             path  int  true  "Email ID"
// @Param        attachment_id  path  int  true  "Attachment ID"
// @Success      200
// @Router       /api/v1/email/emails/{id}/attachments/{attachment_id} [get]
func (h *Handler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	attachmentIDStr := mux.Vars(r)["attachment_id"]
	attachmentID, err := strconv.ParseInt(attachmentIDStr, 10, 64)
	if err != nil || attachmentID <= 0 {
		response.BadRequest(w)
		return
	}

	result, err := h.service.DownloadAttachment(r.Context(), service.DownloadAttachmentInput{
		UserID:       userID,
		EmailID:      emailID,
		AttachmentID: attachmentID,
	})
	if err != nil {
		logger.Errorf("DownloadAttachment: user_id=%d, email_id=%d, attachment_id=%d, err=%v",
			userID, emailID, attachmentID, err)
		parseCommonErrors(err, w)
		return
	}
	defer func() {
		if err := result.Body.Close(); err != nil {
			logger.Errorf("file close error: %v", err)
		}
	}()

	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, result.FileName))
	w.Header().Set("Content-Length", strconv.FormatInt(result.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, result.Body); err != nil {
		logger.Errorf("DownloadAttachment stream error: user_id=%d, attachment_id=%d, err=%v",
			userID, attachmentID, err)
	}
}

// DeleteAttachments godoc
// @Summary      Delete attachments
// @Tags         attachments
// @Accept       json
// @Param        id   path  int        true  "Email ID"
// @Param        body body  IDsRequest true  "Attachment IDs"
// @Success      204
// @Router       /api/v1/email/emails/{id}/attachments [delete]
func (h *Handler) DeleteAttachments(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}

	emailID, err := parsePathInt64(r, "id")
	if err != nil || emailID <= 0 {
		response.BadRequest(w)
		return
	}

	ids, ok := decodeIDs(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteAttachments(r.Context(), service.DeleteAttachmentsInput{
		UserID:        userID,
		EmailID:       emailID,
		AttachmentIDs: ids,
	}); err != nil {
		logger.Errorf("DeleteAttachments: user_id=%d, email_id=%d, ids=%v, err=%v",
			userID, emailID, ids, err)
		parseCommonErrors(err, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func attachmentToResponse(a service.AttachmentResult) AttachmentResponse {
	return AttachmentResponse{
		ID:          a.ID,
		EmailID:     a.EmailID,
		FileName:    a.FileName,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		CreatedAt:   a.CreatedAt,
	}
}
