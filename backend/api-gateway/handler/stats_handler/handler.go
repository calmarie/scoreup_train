package stats_handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
)

type Handler struct {
	statsClient statspb.StatsClient
	logger      *slog.Logger
}

func NewHandler(statsClient statspb.StatsClient, logger *slog.Logger) *Handler {
	return &Handler{statsClient: statsClient, logger: logger}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/stats/stats", h.getStats)
	mux.HandleFunc("GET /api/v1/stats/xp", h.getXP)
	mux.HandleFunc("GET /api/v1/stats/streak", h.getCurrentStreak)
	mux.HandleFunc("GET /api/v1/stats/correctstreak", h.getCurrentCorrectStreak)
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetStats(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetStats)
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
		AccuracyPercent:      reply.AccuracyPercent,
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
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetXP)
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
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetCurrentStreak)
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
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetCurrentCorrectStreak)
		return
	}
	h.writeJSON(w, http.StatusOK, CorrectStreak{CurrentCorrectStreak: reply.CurrentCorrectStreak})
}

func (h *Handler) statsRequest(w http.ResponseWriter, r *http.Request) *statspb.UserStatsRequest {
	userID, ok := r.Context().Value("userID").(int64)
	if !ok || userID <= 0 {
		h.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return nil
	}
	return &statspb.UserStatsRequest{UserId: userID}
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		h.logger.Error("write JSON response", "error", err)
	}
}
