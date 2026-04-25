package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/support/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type Service interface {
	SendQuestion(ctx context.Context, cmd service.SendQuestionInput) (string, error)
	GetMyQuestions(ctx context.Context, cmd service.GetMyQuestionsInput) (*service.GetMyQuestionsResult, error)

	ChangeStatus(ctx context.Context, cmd service.ChangeStatusInput) error
	AnswerOnQuestion(ctx context.Context, cmd service.AnswerOnQuestionInput) error

	GetAllMessages(ctx context.Context, cmd service.GettAllMessagesInput) (*service.GettAllMessagesResult, error)

	GetAllQuestionsByFilter(ctx context.Context, cmd service.GetAllQuestionsByFilterInput) (*service.GetAllQuestionsByFilterResult, error)
}

type SendQuestionRequest struct {
	Theme   string `json:"theme"`
	Header  string `json:"header"`
	Quesion string `json:"quesion_text"`
}

func (handler *Handler) SendQuestion(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}

	var req SendQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Errorf("failed to decode request: %v", err)
		response.BadRequest(w)
		return
	}

	if len(req.Quesion) <= 0 || len(req.Theme) <= 0 {
		response.BadRequest(w)
		return
	}

	question, err := handler.service.SendQuestion(r.Context(), service.SendQuestionInput{
		UserID: claims.UserId,
		Theme:  req.Theme,
		Header: req.Header,
		Body:   req.Quesion,
	})
	if err != nil {
		logger.Errorf("failed to send question by %d: %v", claims.UserId, err)
		parseCommonErrors(err, w)
		return
	}

	logger.Infof("question send %d", claims.UserId)

	if err := json.NewEncoder(w).Encode(map[string]string{"question": question}); err != nil {
		response.InternalError(w)
	}
}

type GetMyQuestionResponse struct {
	TickerID int64  `json:"ticket_id"`
	Status   string `json:"status"`
	Theme    string `json:"theme"`
	Header   string `json:"header"`
}

func (handler *Handler) GetMyQuestions(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}

	result, err := handler.service.GetMyQuestions(r.Context(), service.GetMyQuestionsInput{
		UserID: claims.UserId,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	questions := make([]GetMyQuestionResponse, len(result.Questions))
	for i, question := range result.Questions {
		questions[i] = GetMyQuestionResponse{
			TickerID: question.TickerID,
			Status:   question.Status,
			Theme:    question.Theme,
			Header:   question.Header,
		}
	}

	if err := json.NewEncoder(w).Encode(questions); err != nil {
		response.InternalError(w)
		return
	}
}

type ChangeStatusRequest struct {
	Status     string `json:"status"`
	QuestionID int64  `json:"question_id"`
}

func (handler *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}

	var req ChangeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	err = handler.service.ChangeStatus(r.Context(), service.ChangeStatusInput{
		UserID:     claims.UserId,
		Status:     req.Status,
		QuestionID: req.QuestionID,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		response.InternalError(w)
		return
	}
}

type AnswerOnQuestionRequest struct {
	QuestionID int64  `json:"question_id"`
	Answer     string `json:"answer"`
}

func (handler *Handler) AnswerOnQuestion(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}

	var req AnswerOnQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w)
		return
	}

	err = handler.service.AnswerOnQuestion(r.Context(), service.AnswerOnQuestionInput{
		UserID:     claims.UserId,
		QuestionID: req.QuestionID,
		Answer:     req.Answer,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		response.InternalError(w)
		return
	}
}

type GetAllQuestionsByFilterRequest struct {
	Status string `json:"status"`
	Theme  string `json:"theme"`
}
type GetAllQuestionsByFilterResponse struct {
	TickerID int64  `json:"ticket_id"`
	Status   string `json:"status"`
	Theme    string `json:"theme"`
	Header   string `json:"header"`
}

func (handler *Handler) GetAllQuestionsByFilter(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}

	var req GetAllQuestionsByFilterRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warnf("Invalid request body: %v", err)
			response.BadRequest(w)
			return
		}
	}

	result, err := handler.service.GetAllQuestionsByFilter(r.Context(), service.GetAllQuestionsByFilterInput{
		UserID: claims.UserId,
		Status: req.Status,
		Theme:  req.Theme,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	questions := make([]GetAllQuestionsByFilterResponse, len(result.Questions))
	for i, question := range result.Questions {
		questions[i] = GetAllQuestionsByFilterResponse{
			TickerID: question.TickerID,
			Status:   question.Status,
			Theme:    question.Theme,
			Header:   question.Header,
		}
	}

	if err := json.NewEncoder(w).Encode(questions); err != nil {
		response.InternalError(w)
		return
	}
}

type GetAllMessagesRequest struct {
	QuestionID int64  `json:"question_id"`
	Answer     string `json:"answer"`
}
type GetAllMessagesResponse struct {
	IsAdmin bool   `json:"is_admin"`
	Text    string `json:"text"`
}

func (handler *Handler) GetAllMessages(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r.Context())

	claims, err := middleware.ClaimsFromContext(r.Context())
	if err != nil {
		logger.Errorf("failed to get claims from context: %v", err)
		response.InternalError(w)
		return
	}

	if claims.UserId <= 0 {
		logger.Warnf("Invalid user ID in claims: %d", claims.UserId)
		response.BadRequest(w)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		logger.Warnf("Invalid url %v", err)
		response.BadRequest(w)
		return
	}

	questionIDstr := pathParts[4]
	questionID, err := strconv.ParseInt(questionIDstr, 10, 64)
	if err != nil {
		logger.Warnf("Invalid question ID format: %s", questionIDstr)
		response.BadRequest(w)
		return
	}
	result, err := handler.service.GetAllMessages(r.Context(), service.GettAllMessagesInput{
		UserID:     claims.UserId,
		QuestionID: questionID,
	})
	if err != nil {
		parseCommonErrors(err, w)
		return
	}

	messages := make([]GetAllMessagesResponse, len(result.Messages))
	for i, message := range result.Messages {
		messages[i] = GetAllMessagesResponse{
			IsAdmin: message.IsAdmin,
			Text:    message.Text,
		}
	}

	if err := json.NewEncoder(w).Encode(messages); err != nil {
		response.InternalError(w)
		return
	}
}
