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
	log.Printf("Redis cache initialized successfully at %s", redisURL)
}

// Close closes the Redis connection
func Close() {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		}
		log.Println("Redis connection closed")
	}
}

// Set stores a value in cache with expiration
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !enabled {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("Error marshaling data for cache key %s: %v", key, err)
		return err
	}

	if err := redisClient.Set(ctx, key, data, expiration).Err(); err != nil {
		log.Printf("Error setting cache key %s: %v", key, err)
		return err
	}

	return nil
}

// Get retrieves a value from cache
func Get(ctx context.Context, key string, dest interface{}) error {
	if !enabled {
		return redis.Nil
	}

	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Error getting cache key %s: %v", key, err)
		}
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		log.Printf("Error unmarshaling data from cache key %s: %v", key, err)
		return err
	}

	return nil
}

// Delete removes a key from cache
func Delete(ctx context.Context, key string) error {
	if !enabled {
		return nil
	}

	if err := redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Error deleting cache key %s: %v", key, err)
		return err
	}

	return nil
}

// BatchDelete removes multiple keys from cache
func BatchDelete(ctx context.Context, keys ...string) error {
	if !enabled || len(keys) == 0 {
		return nil
	}

	if err := redisClient.Del(ctx, keys...).Err(); err != nil {
		log.Printf("Error batch deleting cache keys: %v", err)
		return err
	}

	return nil
}