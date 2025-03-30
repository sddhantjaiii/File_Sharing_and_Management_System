package utils

import (
	"context"
	"encoding/json"
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

func SetCache(key string, value interface{}, expiration time.Duration) error {
	ctx := context.Background()
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, jsonValue, expiration).Err()
}

func GetCache(key string, value interface{}) error {
	ctx := context.Background()
	jsonValue, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(jsonValue), value)
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