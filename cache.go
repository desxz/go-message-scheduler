package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheConfig struct {
	TTL time.Duration `mapstructure:"ttl"`
}

type RedisCache struct {
	client *redis.Client
	config CacheConfig
}

func NewRedisCache(addr, password string, db int, config CacheConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client: client,
		config: config,
	}
}

func (c *RedisCache) Set(ctx context.Context, key string, value string) error {
	return c.client.Set(ctx, key, value, c.config.TTL).Err()
}
