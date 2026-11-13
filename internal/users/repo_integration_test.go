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

func TestRepository_GetUserByTelegramID(t *testing.T) {
	ctx := context.Background()

	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load env file: %v", err)
	}

	databaseURL, err := config.DbURL("POSTGRES_DB_TEST")
	if err != nil {
		t.Fatalf("read postgres configuration: %v", err)
	}

	db, err := postgres.Connect(ctx, databaseURL+"&pool_max_conns=1")
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer db.Close()

	migration, err := os.ReadFile("../../migrations/000002_create_users.up.sql")
	if err != nil {
		t.Fatalf("read users migration: %v", err)
	}

	query := strings.Replace(string(migration), "CREATE TABLE users", "CREATE TEMP TABLE users", 1)
	if _, err := db.Exec(ctx, query); err != nil {
		t.Fatalf("create temporary users table: %v", err)
	}

	const telegramID int64 = 123456789
	if _, err := db.Exec(ctx, `INSERT INTO users DEFAULT VALUES`); err != nil {
		t.Fatalf("insert user without telegram ID: %v", err)
	}

	var wantID int64
	if err := db.QueryRow(ctx, `
		INSERT INTO users (telegram_id)
		VALUES ($1)
		RETURNING id;`, telegramID).Scan(&wantID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	repo := NewRepository(db)
	user, err := repo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	if user.Id != wantID {
		t.Errorf("id = %d, want %d", user.Id, wantID)
	}
	if user.Telegram_id == nil || *user.Telegram_id != telegramID {
		t.Errorf("telegram_id = %v, want %d", user.Telegram_id, telegramID)
	}
	if user.Created_at.IsZero() {
		t.Error("created_at must contain the creation time")
	}

	_, err = repo.GetUserByTelegramID(ctx, telegramID+1)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want %v", err, ErrUserNotFound)
	}
}
