package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client

// InitRedis initializes the connection to the Redis server
func InitRedis() {
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "localhost:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Println("Connected to Redis Cache")
}

// UpdatePrice pushes the latest price to Redis
func UpdatePrice(symbol string, price float64) error {
	// The Node.js dispatcher expects keys in the format "price:SYMBOL"
	// Example: price:BTC -> 64000.5
	key := fmt.Sprintf("price:%s", symbol)
	val := fmt.Sprintf("%f", price)

	err := rdb.Set(ctx, key, val, 0).Err() // 0 means no expiration currently, or we can set TTL
	if err != nil {
		return err
	}
	
	return nil
}
