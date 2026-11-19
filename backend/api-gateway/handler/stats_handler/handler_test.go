package stats_handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statspb "github.com/calmarie/scoreup_train/backend/gen/stats"
	userpb "github.com/calmarie/scoreup_train/backend/gen/user"
	"google.golang.org/genproto/googleapis/type/date"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type testConn struct {
	grpc.ClientConnInterface
	invoke func(method string, req, reply any) error
}

func (c testConn) Invoke(_ context.Context, method string, req, reply any, _ ...grpc.CallOption) error {
	return c.invoke(method, req, reply)
}

func TestStatsRoutesLookUpTelegramUser(t *testing.T) {
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
			users := userpb.NewUserClient(testConn{invoke: func(method string, req, reply any) error {
				if method != userpb.User_GetUserByTelegramID_FullMethodName || req.(*userpb.GetUserByTelegramIDRequest).TelegramId != 123 {
					t.Fatalf("unexpected user lookup: %s %v", method, req)
				}
				reply.(*userpb.UserReply).Id = 42
				return nil
			}})
			stats := statspb.NewStatsClient(testConn{invoke: func(method string, req, reply any) error {
				if method != tc.method || req.(*statspb.UserStatsRequest).UserId != 42 {
					t.Fatalf("unexpected stats call: %s %v", method, req)
				}
				proto.Merge(reply.(proto.Message), tc.reply)
				return nil
			}})
			mux := http.NewServeMux()
			NewHandler(users, stats, slog.New(slog.DiscardHandler)).RegisterRoutes(mux)
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/stats/"+tc.path+"/123", nil))
			if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != tc.want {
				t.Fatalf("status = %d; body = %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestGetXPStopsWhenUserLookupFails(t *testing.T) {
	for _, tc := range []struct {
		code       codes.Code
		wantStatus int
		wantBody   string
	}{
		{codes.NotFound, http.StatusNotFound, `{"error":"user not found"}`},
		{codes.InvalidArgument, http.StatusBadRequest, `{"error":"invalid telegram id"}`},
	} {
		t.Run(tc.code.String(), func(t *testing.T) {
			users := userpb.NewUserClient(testConn{invoke: func(string, any, any) error {
				return status.Error(tc.code, "upstream detail")
			}})
			mux := http.NewServeMux()
			NewHandler(users, nil, slog.New(slog.DiscardHandler)).RegisterRoutes(mux)

			res := httptest.NewRecorder()
			mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/stats/xp/123", nil))
			if res.Code != tc.wantStatus || strings.TrimSpace(res.Body.String()) != tc.wantBody {
				t.Fatalf("status = %d; body = %s", res.Code, res.Body.String())
			}
		})
	}
}
