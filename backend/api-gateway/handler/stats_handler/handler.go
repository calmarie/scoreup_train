package stats_handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	"google.golang.org/genproto/googleapis/type/date"
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
	mux.HandleFunc("GET /api/v1/stats/daily/{date}", h.getDailyStats)
	mux.HandleFunc("GET /api/v1/stats/weekly", h.getWeeklyStats)
	mux.HandleFunc("GET /api/v1/stats/topics/{topic}", h.getTopicStats)
	mux.HandleFunc("GET /api/v1/stats/topics", h.getAllTopicStats)
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

func (h *Handler) getDailyStats(w http.ResponseWriter, r *http.Request) {
	user := h.statsRequest(w, r)
	if user == nil {
		return
	}
	day, err := time.Parse(time.DateOnly, r.PathValue("date"))
	if err != nil || day.Year() < 1 {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date must be YYYY-MM-DD"})
		return
	}
	reply, err := h.statsClient.GetDailyStats(r.Context(), &statspb.GetDailyStatsRequest{
		UserId: user.UserId,
		Date:   &date.Date{Year: int32(day.Year()), Month: int32(day.Month()), Day: int32(day.Day())},
	})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetStats)
		return
	}
	h.writeJSON(w, http.StatusOK, dailyStatsJSON(reply))
}

func (h *Handler) getWeeklyStats(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetWeeklyStats(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetStats)
		return
	}
	days := make([]DailyStats, 0, len(reply.GetDays()))
	for _, day := range reply.GetDays() {
		days = append(days, dailyStatsJSON(day))
	}
	h.writeJSON(w, http.StatusOK, WeeklyStats{Days: days})
}

func (h *Handler) getTopicStats(w http.ResponseWriter, r *http.Request) {
	user := h.statsRequest(w, r)
	if user == nil {
		return
	}
	reply, err := h.statsClient.GetTopicStats(r.Context(), &statspb.GetTopicStatsRequest{
		UserId: user.UserId,
		Topic:  r.PathValue("topic"),
	})
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetStats)
		return
	}
	h.writeJSON(w, http.StatusOK, topicStatsJSON(reply))
}

func (h *Handler) getAllTopicStats(w http.ResponseWriter, r *http.Request) {
	req := h.statsRequest(w, r)
	if req == nil {
		return
	}
	reply, err := h.statsClient.GetAllTopicStats(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, r, decodeGRPCError(err), ErrGetStats)
		return
	}
	topics := make([]TopicStats, 0, len(reply.GetTopics()))
	for _, topic := range reply.GetTopics() {
		topics = append(topics, topicStatsJSON(topic))
	}
	h.writeJSON(w, http.StatusOK, AllTopicStats{Topics: topics})
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
