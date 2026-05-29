package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
)

// @Summary      Ответить на анонимное письмо
// @Description  Отправляет ответ на анонимное письмо. Получатель — автор родительского письма.
// @Description  Если is_anonymous=true, получатель не увидит email отправителя.
// @Tags         emails
// @Accept       json
// @Accept       multipart/form-data
// @Produce      json
// @Param        id              path      int                  true  "ID родительского письма"
// @Param        body            body      handler.ReplyRequest false "Тело запроса (JSON)"
// @Param        header          formData  string               false "Тема письма (multipart)"
// @Param        body            formData  string               false "Тело письма (multipart)"
// @Param        is_anonymous    formData  bool                 false "Отправить анонимно (multipart)"
// @Param        attachments     formData  file                 false "Вложения (multipart)"
// @Success      200             {object}  handler.SendEmailResponse
// @Failure      400             {object}  response.ErrorResponse  "Пустое тело или некорректный запрос"
// @Failure      403             {object}  response.ErrorResponse  "Нет доступа к родительскому письму"
// @Failure      404             {object}  response.ErrorResponse  "Родительское письмо не найдено"
// @Failure      422             {object}  response.ErrorResponse  "Родитель не анонимный или отправитель — автор оригинала"
// @Failure      500             {object}  response.ErrorResponse
// @Security     ApiKeyAuth
// @Router       /emails/{id}/reply [post]
func (h *Handler) Reply(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())
	userID, ok := userIDFromCtx(r, w)
	if !ok {
		return
	}
	parentID, err := parsePathInt64(r, "id")
	if err != nil || parentID <= 0 {
		response.BadRequest(w)
		return
	}

	ct := r.Header.Get("Content-Type")
	in := service.ReplyInput{UserID: userID, ParentEmailID: parentID}

	if isMultipart(ct) {
		if err := r.ParseMultipartForm(maxSendFormSize); err != nil {
			response.BadRequest(w)
			return
		}
		in.Header = r.FormValue("header")
		in.Body = r.FormValue("body")
		if v := r.FormValue("is_anonymous"); v != "" {
			if parsed, err := strconv.ParseBool(v); err == nil {
				in.IsAnonymous = parsed
			}
		}
		if r.MultipartForm != nil {
			for _, fhs := range r.MultipartForm.File {
				for _, fh := range fhs {
					f, err := fh.Open()
					if err != nil {
						response.BadRequest(w)
						return
					}
					in.Files = append(in.Files, f)
					in.FileHeaders = append(in.FileHeaders, fh)
				}
			}
		}
	} else {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			response.BadRequest(w)
			return
		}
		var req ReplyRequest
		if err := req.UnmarshalJSON(body); err != nil {
			response.BadRequest(w)
			return
		}
		in.Header = req.Header
		in.Body = req.Body
		in.IsAnonymous = req.IsAnonymous
	}

	if in.Header == "" && in.Body == "" {
		response.BadRequest(w)
		return
	}

	result, err := h.service.Reply(r.Context(), in)
	if err != nil {
		logger.Errorf("Reply: user_id=%d, parent_id=%d, err=%v", userID, parentID, err)
		parseCommonErrors(err, w)
		return
	}

	resp := SendEmailResponse{
		ID:            result.ID,
		SenderID:      result.SenderID,
		Header:        result.Header,
		Body:          result.Body,
		IsAnonymous:   result.IsAnonymous,
		ParentEmailID: &parentID,
		CreatedAt:     result.CreatedAt,
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
