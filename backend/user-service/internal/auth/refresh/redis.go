package refresh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRefreshTokenInvalid = errors.New("refresh token is invalid or expired")

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func (s *RedisStore) Save(ctx context.Context, hash string, userID int64) error {
	if err := s.rdb.Set(ctx, "refresh:"+hash, userID, 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("set refresh token: %w", err)
	}
	return nil
}

func (s *RedisStore) Get(ctx context.Context, hash string) (int64, error) {
	userID, err := s.rdb.Get(ctx, "refresh:"+hash).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, ErrRefreshTokenInvalid
	}
	if err != nil {
		return 0, fmt.Errorf("get refresh token: %w", err)
	}
	return userID, nil
}

func New() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	return rdb
}

func CheckConn(ctx context.Context, rdb *redis.Client) error {
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Connect redis error: %w", err)
	}
	return nil
}
