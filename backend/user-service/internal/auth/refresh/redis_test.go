package refresh

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisStoreReportsFailedWrite(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	if err := NewRedisStore(client).Save(context.Background(), "hash", 42); err == nil {
		t.Fatal("expected connection error from Redis write")
	}
}

func TestCheckConnReportsUnavailableRedis(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	if err := CheckConn(context.Background(), client); err == nil {
		t.Fatal("expected Redis ping error")
	}
}
