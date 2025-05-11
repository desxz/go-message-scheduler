package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupRedisContainer starts a Redis container for testing and returns the container and connection URL
func setupRedisContainer(t *testing.T) (testcontainers.Container, string) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get the Redis connection string
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	connectionURL := fmt.Sprintf("%s:%s", host, port.Port())
	return container, connectionURL
}

func TestRedisCache_Set(t *testing.T) {
	ctx := context.Background()

	container, redisURL := setupRedisContainer(t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	defer client.Close()

	tests := []struct {
		name    string
		key     string
		value   string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "set with default TTL",
			key:     "test-key-1",
			value:   "test-value-1",
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "set with short TTL",
			key:     "test-key-2",
			value:   "test-value-2",
			ttl:     5 * time.Second,
			wantErr: false,
		},
		{
			name:    "set with long TTL",
			key:     "test-key-3",
			value:   "test-value-3",
			ttl:     time.Hour,
			wantErr: false,
		},
		{
			name:    "set with empty key",
			key:     "",
			value:   "test-empty-key",
			ttl:     time.Minute,
			wantErr: false,
		},
		{
			name:    "set with empty value",
			key:     "empty-value",
			value:   "",
			ttl:     time.Minute,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &RedisCache{
				client: client,
				config: CacheConfig{TTL: tt.ttl},
			}

			err := cache.Set(ctx, tt.key, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			value, err := client.Get(ctx, tt.key).Result()
			assert.NoError(t, err)
			assert.Equal(t, tt.value, value)

			ttl, err := client.TTL(ctx, tt.key).Result()
			assert.NoError(t, err)
			assert.True(t, ttl > 0 && ttl <= tt.ttl+time.Second)
		})
	}
}
