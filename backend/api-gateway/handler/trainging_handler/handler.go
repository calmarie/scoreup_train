package training_handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	trainingpb "github.com/calmarie/scoreup_train/backend/gen/training"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
)

type Handler struct {
	trainingClient trainingpb.TrainingClient
	userClient     userpb.UserClient
	logger         *slog.Logger
}

const maxRequestBodyBytes int64 = 1 << 20

func NewHandler(trainingClient trainingpb.TrainingClient, userClient userpb.UserClient, logger *slog.Logger) *Handler {
	return &Handler{trainingClient: trainingClient, userClient: userClient, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/question", h.getQuestion)
	mux.HandleFunc("POST /api/v1/question/answer", h.saveAnswer)
}

func (h *Handler) getQuestion(w http.ResponseWriter, r *http.Request) {
	reply, err := h.trainingClient.GetQuestion(r.Context(), &trainingpb.GetQuestionRequest{})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrQuestionNotFound), ErrGetQuestion)
		return
	}

	question := Question{
		ID:          int(reply.Id),
		Type:        reply.Type,
		Difficulty:  int(reply.Difficulty),
		Topic:       reply.Topic,
		Skill:       reply.Skill,
		Text:        reply.Text,
		ImageURL:    reply.ImageUrl,
		Options:     reply.Options,
		Explanation: reply.Explanation,
		Data:        reply.Data,
		IsActive:    reply.IsActive,
	}
	h.writeJSON(w, http.StatusOK, question)
}

func (h *Handler) saveAnswer(w http.ResponseWriter, r *http.Request) {
	var body SubmitAnswerRequest
	if err := decodeJSON(w, r, &body); err != nil {
		h.handleServiceError(w, r, err, ErrSaveAnswer)
		return
	}
	if body.TelegramID <= 0 {
		h.handleServiceError(w, r, ErrInvalidTelegramID, ErrSaveAnswer)
		return
	}
	if body.QuestionID <= 0 {
		h.handleServiceError(w, r, ErrInvalidQuestionID, ErrSaveAnswer)
		return
	}
	if len(body.Answer) == 0 {
		h.handleServiceError(w, r, ErrAnswerRequired, ErrSaveAnswer)
		return
	}

	user, err := h.userClient.GetUserByTelegramID(r.Context(), &userpb.GetUserByTelegramIDRequest{
		TelegramId: body.TelegramID,
	})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrUserNotFound), ErrGetUser)
		return
	}

	reply, err := h.trainingClient.SaveAnswer(r.Context(), &trainingpb.SaveAnswerRequest{
		UserId:     user.Id,
		QuestionId: body.QuestionID,
		AnswerJson: body.Answer,
	})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrQuestionNotFound), ErrSaveAnswer)
		return
	}

	answer := Answer{
		ID:        reply.Id,
		IsCorrect: reply.IsCorrect,
		CreatedAt: reply.CreatedAt.AsTime(),
	}

	awardReq := &userpb.AwardQuestionXPRequest{
		UserId:     user.Id,
		QuestionId: body.QuestionID,
		Xp:         reply.Xp,
	}

	h.userClient.AwardQuestionXP(r.Context(), awardReq)

	h.writeJSON(w, http.StatusCreated, answer)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return requestBodyError(err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err != nil {
			return requestBodyError(err)
		}
		return ErrMultipleJSONValues
	}
	return nil
}

func requestBodyError(err error) error {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		return fmt.Errorf("%w: %w", ErrRequestBodyTooLarge, err)
	}
	return fmt.Errorf("%w: %w", ErrInvalidRequestBody, err)
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write JSON response", "error", err)
	}
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, fallback error) {
	status, message := http.StatusInternalServerError, fallback.Error()
	switch {
	case errors.Is(err, context.Canceled):
		status, message = http.StatusRequestTimeout, "request canceled"
	case errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusGatewayTimeout, "request timed out"
	case errors.Is(err, ErrRequestBodyTooLarge):
		status, message = http.StatusRequestEntityTooLarge, ErrRequestBodyTooLarge.Error()
	case errors.Is(err, ErrInvalidRequestBody):
		status, message = http.StatusBadRequest, ErrInvalidRequestBody.Error()
	case errors.Is(err, ErrMultipleJSONValues):
		status, message = http.StatusBadRequest, ErrMultipleJSONValues.Error()
	case errors.Is(err, ErrInvalidTelegramID):
		status, message = http.StatusBadRequest, ErrInvalidTelegramID.Error()
	case errors.Is(err, ErrInvalidQuestionID):
		status, message = http.StatusBadRequest, ErrInvalidQuestionID.Error()
	case errors.Is(err, ErrAnswerRequired):
		status, message = http.StatusBadRequest, ErrAnswerRequired.Error()
	case errors.Is(err, ErrUserNotFound):
		status, message = http.StatusNotFound, ErrUserNotFound.Error()
	case errors.Is(err, ErrQuestionNotFound):
		status, message = http.StatusNotFound, ErrQuestionNotFound.Error()
	case errors.Is(err, ErrServiceUnavailable):
		status, message = http.StatusServiceUnavailable, ErrServiceUnavailable.Error()
	}
	if status >= http.StatusInternalServerError {
		h.logger.ErrorContext(r.Context(), "training request failed",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}
	h.writeJSON(w, status, map[string]string{"error": message})
}
