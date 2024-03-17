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
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password set
		DB:       0,                // Use default DB
	})

	// Example: Caching data
	// key := "somekey"
	// data, err := getDataFromCache(rdb, key)
	// if err != nil {
	//     // If data not found in cache, retrieve it and cache it
	//     data = fetchDataFromDatabase()

	//     // Cache the data for future use
	//     err := cacheData(rdb, key, data, 10*time.Minute) // Cache data for 10 minutes
	//     if err != nil {
	//         fmt.Println("Error caching data:", err)
	//     }
	// }

	// fmt.Println("Data:", data)

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
