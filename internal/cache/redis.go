package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	enabled     bool
)

// Initialize sets up Redis connection if REDIS_URL is provided
func Initialize(redisURL string) {
	if redisURL == "" {
		log.Println("Redis URL not provided, caching disabled")
		enabled = false
		return
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL: %v, caching disabled", err)
		enabled = false
		return
	}

	redisClient = redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v, caching disabled", err)
		enabled = false
		return
	}

	enabled = true
	log.Println("Redis cache initialized successfully")
}

// Close closes the Redis connection
func Close() {
	if redisClient != nil {
		redisClient.Close()
	}
}

// Set stores a value in cache with expiration
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value from cache
func Get(ctx context.Context, key string, dest interface{}) error {
	if !enabled {
		return redis.Nil
	}

	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Delete removes a key from cache
func Delete(ctx context.Context, key string) error {
	if !enabled {
		return nil
	}

	return redisClient.Del(ctx, key).Err()
}
