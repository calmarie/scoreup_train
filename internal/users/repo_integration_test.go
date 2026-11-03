package users

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/calmarie/scoreup_train/internal/config"
	"github.com/calmarie/scoreup_train/internal/postgres"
	"github.com/joho/godotenv"
)

func TestRepository_CreateUser_Success(t *testing.T) {
	ctx := context.Background()

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load env file: %v", err)
	}

	databaseURL, err := config.DbURL("POSTGRES_DB_TEST")
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}

	// A single connection keeps the temporary table available to the repository.
	db, err := postgres.Connect(ctx, databaseURL+"&pool_max_conns=1")
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	migration, err := os.ReadFile("../../migrations/000002_create_users.up.sql")
	if err != nil {
		t.Fatalf("read users migration: %v", err)
	}

	// The temporary table uses the migration schema and disappears on disconnect.
	query := strings.Replace(string(migration), "CREATE TABLE users", "CREATE TEMP TABLE users", 1)
	if _, err := db.Exec(ctx, query); err != nil {
		t.Fatalf("create temporary users table: %v", err)
	}

	repo := NewRepository(db)
	for id := int64(1); id <= 2; id++ {
		user, err := repo.CreateUser(ctx)
		if err != nil {
			t.Fatalf("call CreateUser: %v", err)
		}

		if user.Id != id {
			t.Errorf("id = %d, want %d", user.Id, id)
		}

		if user.Telegram_id != nil {
			t.Errorf("telegram_id = %d, want nil", *user.Telegram_id)
		}

		if user.Created_at.IsZero() {
			t.Error("created_at must contain the creation time")
		}
	}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.CreateUser(canceledCtx); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want %v", err, context.Canceled)
	}

	var count int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count created users: %v", err)
	}
	if count != 2 {
		t.Errorf("users count = %d, want %d", count, 2)
	}
}
