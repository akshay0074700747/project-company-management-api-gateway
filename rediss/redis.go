package rediss

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{
		Client: client,
	}
}

func NewRedis() *redis.Client {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-service:6379",
		Password: "",
		DB:       0,
	})


	return rdb
}

func (cache *Cache) GetDataFromCache(key string, val interface{}, ctx context.Context) error {
	err := cache.Client.Get(ctx, key).Scan(val)
	return err
}

func (cache *Cache) CacheData(key string, data []byte, expiration time.Duration, ctx context.Context) error {

	err := cache.Client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}
