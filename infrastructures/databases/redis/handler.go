package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	"time"

	goRedis "github.com/redis/go-redis/v9"
)

// RedisInstance wraps a single Redis connection. Create one per Redis the service needs, e.g.
//
//	mainRedis := redis.NewRedisInstance(configs, configs.Envs.MainRedis)
//	sessionRedis := redis.NewRedisInstance(configs, configs.Envs.SessionRedis)
type RedisInstance struct {
	// Client is exposed for operations beyond caching (pub/sub, counters, locks, etc.). Keys used here are not prefixed
	Client     *goRedis.Client
	Name       string
	KeyPrefix  string
	DefaultTTL time.Duration
}

func NewRedisInstance(configs *configurations.Configs, redisConfig models.RedisConfig) *RedisInstance {
	options, err := goRedis.ParseURL(redisConfig.URL)
	if err != nil {
		configs.Logger.Fatal("Invalid Redis URL", "name", redisConfig.Name, "error", err)
	}

	if redisConfig.PoolSize > 0 {
		options.PoolSize = redisConfig.PoolSize
	}

	client := goRedis.NewClient(options)

	ctx := context.Background()
	if redisConfig.PingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, redisConfig.PingTimeout)
		defer cancel()
	}

	if err := client.Ping(ctx).Err(); err != nil {
		configs.Logger.Fatal("Redis connection failed", "name", redisConfig.Name, "addr", options.Addr, "error", err)
	}

	configs.Logger.Info("Redis Successfully Connected", "name", redisConfig.Name, "addr", options.Addr, "db", options.DB)

	return &RedisInstance{
		Client:     client,
		Name:       redisConfig.Name,
		KeyPrefix:  redisConfig.KeyPrefix,
		DefaultTTL: redisConfig.DefaultTTL,
	}
}

func (r *RedisInstance) buildKey(key string) string {
	return r.KeyPrefix + key
}

// SaveCache stores value as JSON. ttl <= 0 uses DefaultTTL, and a zero DefaultTTL means the key never expires
func (r *RedisInstance) SaveCache(ctx context.Context, key string, value any, ttl time.Duration) error {
	assignedKey := r.buildKey(key)

	if ttl <= 0 {
		ttl = r.DefaultTTL
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis %s save cache %q: marshal: %w", r.Name, assignedKey, err)
	}

	if err := r.Client.Set(ctx, assignedKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis %s save cache %q: %w", r.Name, assignedKey, err)
	}

	return nil
}

// GetCache decodes the cached JSON into dest, which must be a pointer. Returns models.ErrCacheMiss when the key is not found
func (r *RedisInstance) GetCache(ctx context.Context, key string, dest any) error {
	assignedKey := r.buildKey(key)

	data, err := r.Client.Get(ctx, assignedKey).Bytes()
	if errors.Is(err, goRedis.Nil) {
		return models.ErrCacheMiss
	}

	if err != nil {
		return fmt.Errorf("redis %s get cache %q: %w", r.Name, assignedKey, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("redis %s get cache %q: unmarshal: %w", r.Name, assignedKey, err)
	}

	return nil
}

// RemoveCache deletes one or more keys. Missing keys are ignored
func (r *RedisInstance) RemoveCache(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	assignedKeys := make([]string, len(keys))
	for i, key := range keys {
		assignedKeys[i] = r.buildKey(key)
	}

	if err := r.Client.Del(ctx, assignedKeys...).Err(); err != nil {
		return fmt.Errorf("redis %s remove cache %v: %w", r.Name, assignedKeys, err)
	}

	return nil
}

func (r *RedisInstance) Close() error {
	return r.Client.Close()
}
