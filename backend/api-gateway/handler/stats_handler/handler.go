package stats_handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
)

type Handler struct {
	userClient  userpb.UserClient
	statsClient statspb.StatsClient
	logger      *slog.Logger
}

func NewHandler(userClient userpb.UserClient, statsClient statspb.StatsClient, logger *slog.Logger) *Handler {
	return &Handler{userClient: userClient, statsClient: statsClient, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/stats/stats/{telegramID}", h.getStats)
	mux.HandleFunc("GET /api/v1/stats/xp/{telegramID}", h.getXP)
	mux.HandleFunc("GET /api/v1/stats/streak/{telegramID}", h.getCurrentStreak)
	mux.HandleFunc("GET /api/v1/stats/correctstreak/{telegramID}", h.getCurrentCorrectStreak)
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetStats(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrInvalidStatsRequest, ErrStatsNotFound), ErrGetStats)
		return
	}

	var lastSolvedOn *string
	if date := reply.LastSolvedOn; date != nil {
		day := fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
		lastSolvedOn = &day
	}
	h.writeJSON(w, http.StatusOK, Stats{
		UserID:               reply.UserId,
		XP:                   reply.Xp,
		TotalAnswers:         reply.TotalAnswers,
		CorrectAnswers:       reply.CorrectAnswers,
		UniqueSolved:         reply.UniqueSolved,
		ActiveDays:           reply.ActiveDays,
		CurrentStreak:        reply.CurrentStreak,
		MaxStreak:            reply.MaxStreak,
		CurrentCorrectStreak: reply.CurrentCorrectStreak,
		BestCorrectStreak:    reply.BestCorrectStreak,
		LastSolvedOn:         lastSolvedOn,
	})
}

func (h *Handler) getXP(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetXP(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrInvalidStatsRequest, ErrStatsNotFound), ErrGetXP)
		return
	}
	h.writeJSON(w, http.StatusOK, XP{XP: reply.Xp})
}

func (h *Handler) getCurrentStreak(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetCurrentStreak(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrInvalidStatsRequest, ErrStatsNotFound), ErrGetCurrentStreak)
		return
	}
	h.writeJSON(w, http.StatusOK, Streak{CurrentStreak: reply.CurrentStreak})
}

func (h *Handler) getCurrentCorrectStreak(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetCurrentCorrectStreak(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrInvalidStatsRequest, ErrStatsNotFound), ErrGetCurrentCorrectStreak)
		return
	}
	h.writeJSON(w, http.StatusOK, CorrectStreak{CurrentCorrectStreak: reply.CurrentCorrectStreak})
}

// statsRequest resolves the Telegram ID to the internal user ID used by Stats.
func (h *Handler) statsRequest(w http.ResponseWriter, r *http.Request) *statspb.UserStatsRequest {
	telegramID, err := strconv.ParseInt(r.PathValue("telegramID"), 10, 64)
	if err != nil || telegramID <= 0 {
		h.handleServiceError(w, r, ErrInvalidTelegramID, ErrGetUser)
		return nil
	}

	user, err := h.userClient.GetUserByTelegramID(r.Context(), &userpb.GetUserByTelegramIDRequest{TelegramId: telegramID})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err, ErrInvalidTelegramID, ErrUserNotFound), ErrGetUser)
		return nil
	}
	return &statspb.UserStatsRequest{UserId: user.Id}
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write JSON response", "error", err)
	}
}
