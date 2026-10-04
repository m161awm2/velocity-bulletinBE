package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMiss = errors.New("cache miss")

// Cache is a best-effort cache. Callers should fall back to their source of
// truth when Get or Set returns an error.
type Cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
}

type Redis struct {
	client *redis.Client
}

func NewRedis(rawURL string) (*Redis, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	options.MaxRetries = 0
	options.DialTimeout = 200 * time.Millisecond
	options.ReadTimeout = 200 * time.Millisecond
	options.WriteTimeout = 200 * time.Millisecond
	return &Redis{client: redis.NewClient(options)}, nil
}

func (r *Redis) Ping(ctx context.Context) error { return r.client.Ping(ctx).Err() }

func (r *Redis) Close() error { return r.client.Close() }

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrMiss
	}
	return value, err
}

func (r *Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}
