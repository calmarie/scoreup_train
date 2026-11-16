package solvedquestions

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/calmarie/scoreup_train/backend/user-service/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func newTestRepository(t *testing.T) (*Repository, *pgxpool.Pool, int64) {
	t.Helper()
	ctx := context.Background()
	if err := godotenv.Load("../../../../.env"); err != nil {
		t.Fatalf("load env file: %v", err)
	}
	testUser := os.Getenv("TRAINING_TEST_POSTGRES_USER")
	if testUser == "" {
		testUser = "postgres"
	}
	t.Setenv("POSTGRES_USER", testUser)
	testPassword := os.Getenv("TRAINING_TEST_POSTGRES_PASSWORD")
	if testPassword == "" {
		testPassword = "postgres"
	}
	t.Setenv("POSTGRES_PASSWORD", testPassword)
	testPort := os.Getenv("TRAINING_TEST_POSTGRES_PORT")
	if testPort == "" {
		testPort = "5434"
	}
	t.Setenv("USER_POSTGRES_DB", os.Getenv("POSTGRES_DB_TEST"))
	t.Setenv("USER_POSTGRES_HOST", os.Getenv("POSTGRES_HOST"))
	t.Setenv("USER_POSTGRES_PORT", testPort)

	databaseURL, err := postgres.DatabaseURL()
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}
	db, err := postgres.Connect(ctx, databaseURL+"&pool_max_conns=1")
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(db.Close)

	usersMigration, err := os.ReadFile("../../postgres/migrations/000001_init_schema.up.sql")
	if err != nil {
		t.Fatalf("read users migration: %v", err)
	}
	usersSQL := strings.Replace(string(usersMigration), "CREATE TABLE users", "CREATE TEMP TABLE users", 1)
	if _, err := db.Exec(ctx, usersSQL); err != nil {
		t.Fatalf("create temporary users table: %v", err)
	}
	solvedMigration, err := os.ReadFile("../../postgres/migrations/000002_user_solved_questions.up.sql")
	if err != nil {
		t.Fatalf("read solved questions migration: %v", err)
	}
	solvedSQL := strings.Replace(string(solvedMigration), "CREATE TABLE user_solved_questions", "CREATE TEMP TABLE user_solved_questions", 1)
	if _, err := db.Exec(ctx, solvedSQL); err != nil {
		t.Fatalf("create temporary solved questions table: %v", err)
	}

	var userID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (telegram_id) VALUES (100001) RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return NewRepository(db), db, userID
}

func TestMarkQuestionSolved(t *testing.T) {
	ctx := context.Background()
	repo, db, userID := newTestRepository(t)
	for _, tc := range []struct {
		questionID int
		want       bool
	}{
		{questionID: 7, want: true},
		{questionID: 7, want: false},
		{questionID: 8, want: true},
	} {
		got, err := repo.MarkQuestionSolved(ctx, userID, tc.questionID)
		if err != nil {
			t.Fatalf("MarkQuestionSolved: %v", err)
		}
		if got != tc.want {
			t.Fatalf("MarkQuestionSolved(%d, %d) = %t, want %t", userID, tc.questionID, got, tc.want)
		}
	}

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM user_solved_questions WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("count solved questions: %v", err)
	}
	if count != 2 {
		t.Fatalf("solved questions count = %d, want 2", count)
	}

	var secondUserID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (telegram_id) VALUES (100002) RETURNING id`).Scan(&secondUserID); err != nil {
		t.Fatalf("create second user: %v", err)
	}
	inserted, err := repo.MarkQuestionSolved(ctx, secondUserID, 7)
	if err != nil {
		t.Fatalf("MarkQuestionSolved for second user: %v", err)
	}
	if !inserted {
		t.Fatal("same question should be solvable by another user")
	}
}

func TestAwardQuestionXP(t *testing.T) {
	ctx := context.Background()
	repo, db, userID := newTestRepository(t)
	service := NewService(repo)

	for _, tc := range []struct {
		questionID int
		xp         int
		wantAward  bool
		wantTotal  int
	}{
		{questionID: 7, xp: 100, wantAward: true, wantTotal: 100},
		{questionID: 7, xp: 100, wantAward: false, wantTotal: 100},
		{questionID: 8, xp: 50, wantAward: true, wantTotal: 150},
	} {
		awarded, err := service.AwardQuestionXP(ctx, userID, tc.questionID, tc.xp)
		if err != nil {
			t.Fatalf("AwardQuestionXP: %v", err)
		}
		if awarded != tc.wantAward {
			t.Errorf("awarded = %t, want %t", awarded, tc.wantAward)
		}
		var total int
		if err := db.QueryRow(ctx, `SELECT xp_total FROM users WHERE id = $1`, userID).Scan(&total); err != nil {
			t.Fatalf("get XP total: %v", err)
		}
		if total != tc.wantTotal {
			t.Errorf("XP total = %d, want %d", total, tc.wantTotal)
		}
	}
	if awarded, err := service.AwardQuestionXP(ctx, userID, 9, -1); awarded || !errors.Is(err, ErrNegativeXP) {
		t.Fatalf("negative XP: awarded = %t, error = %v", awarded, err)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE users ADD CONSTRAINT xp_cap CHECK (xp_total <= 150)`); err != nil {
		t.Fatalf("add XP limit for rollback test: %v", err)
	}
	if _, err := service.AwardQuestionXP(ctx, userID, 9, 100); err == nil {
		t.Fatal("expected XP update error")
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM user_solved_questions WHERE user_id = $1 AND question_id = 9`, userID).Scan(&count); err != nil {
		t.Fatalf("count failed solution: %v", err)
	}
	if count != 0 {
		t.Fatalf("failed award left %d solved rows, want 0", count)
	}
}
