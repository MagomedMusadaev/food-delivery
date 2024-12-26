package database

import (
	"context"
	"github.com/redis/go-redis/v9"
)

func InitRedis() (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	_, err := rdb.Ping(context.Background()).Result()

	return rdb, err
}
