package stats_handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type testConn struct {
	grpc.ClientConnInterface
	invoke func(method string, req, reply any) error
}

func (c testConn) Invoke(_ context.Context, method string, req, reply any, _ ...grpc.CallOption) error {
	return c.invoke(method, req, reply)
}

func TestStatsRoutesUseContextUserID(t *testing.T) {
	for _, tc := range []struct {
		path   string
		method string
		reply  proto.Message
		want   string
	}{
		{
			"stats", statspb.Stats_GetStats_FullMethodName,
			&statspb.GetStatsReply{
				UserId: 42, Xp: 90, AccuracyPercent: 57.14, UniqueSolved: 3, ActiveDays: 4,
				CurrentStreak: 2, MaxStreak: 4, CurrentCorrectStreak: 1, BestCorrectStreak: 5,
				LastSolvedOn: &date.Date{Year: 2026, Month: 11, Day: 9},
			},
			`{"user_id":42,"xp":90,"accuracy_percent":57.14,"unique_solved":3,"active_days":4,"current_streak":2,"max_streak":4,"current_correct_streak":1,"best_correct_streak":5,"last_solved_on":"2026-11-09"}`,
		},
		{
			"stats", statspb.Stats_GetStats_FullMethodName, &statspb.GetStatsReply{UserId: 42},
			`{"user_id":42,"xp":0,"accuracy_percent":0,"unique_solved":0,"active_days":0,"current_streak":0,"max_streak":0,"current_correct_streak":0,"best_correct_streak":0,"last_solved_on":null}`,
		},
		{"xp", statspb.Stats_GetXP_FullMethodName, &statspb.GetXPReply{Xp: 90}, `{"xp":90}`},
		{"streak", statspb.Stats_GetCurrentStreak_FullMethodName, &statspb.GetCurrentStreakReply{CurrentStreak: 2}, `{"current_streak":2}`},
		{"correctstreak", statspb.Stats_GetCurrentCorrectStreak_FullMethodName, &statspb.GetCurrentCorrectStreakReply{CurrentCorrectStreak: 1}, `{"current_correct_streak":1}`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			stats := statspb.NewStatsClient(testConn{invoke: func(method string, req, reply any) error {
				if method != tc.method || req.(*statspb.UserStatsRequest).UserId != 42 {
					t.Fatalf("unexpected stats call: %s %v", method, req)
				}
				proto.Merge(reply.(proto.Message), tc.reply)
				return nil
			}})
			mux := http.NewServeMux()
			NewHandler(stats, slog.New(slog.DiscardHandler)).RegisterRoutes(mux)
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/"+tc.path, nil)
			req = req.WithContext(context.WithValue(req.Context(), "userID", int64(42)))
			mux.ServeHTTP(res, req)
			if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != tc.want {
				t.Fatalf("status = %d; body = %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestGetXPRejectsMissingContextUserID(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(nil, slog.New(slog.DiscardHandler)).RegisterRoutes(mux)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/stats/xp", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; body = %s", res.Code, res.Body.String())
	}
}
