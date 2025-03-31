package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func InitRedis() error {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test the connection
	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	return err
}

// GetCache retrieves a string value from Redis cache
func GetCache(key string) (string, error) {
	ctx := context.Background()
	return redisClient.Get(ctx, key).Result()
}

// SetCache sets a string value in Redis cache with expiration
func SetCache(key string, value string, expiration time.Duration) error {
	ctx := context.Background()
	return redisClient.Set(ctx, key, value, expiration).Err()
}

func DeleteCache(key string) error {
	ctx := context.Background()
	return redisClient.Del(ctx, key).Err()
}

func ClearUserCache(userID uint) error {
	ctx := context.Background()
	pattern := "user:*"
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		if err := redisClient.Del(ctx, key).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}
