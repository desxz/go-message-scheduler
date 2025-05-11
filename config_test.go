package main

import (
	"testing"
	"time"

	"github.com/desxz/go-message-scheduler/client"
	"github.com/stretchr/testify/assert"
)

func TestConfig_New(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		configEnv  string
		want       *Config
		wantErr    bool
	}{
		{
			name:       "should return config when valid config file is provided",
			configPath: ".config",
			configEnv:  "dev",
			want: &Config{
				Worker: WorkerConfig{
					WorkerJobInterval: 5 * time.Second,
				},
				WebhookClient: client.WebhookClientConfig{
					Timeout: 30 * time.Second,
					Path:    "/a4d12c37-21b5-4470-92ad-357329f2b48c",
					Host:    "https://webhook.site",
				},
				Cache: CacheConfig{
					TTL: 24 * time.Hour,
				},
				Pool: PoolConfig{
					NumWorkers:      2,
					Timeout:         10 * time.Second,
					InitialJobFetch: true,
				},
				RateLimiter: RateLimiterConfig{
					MaxTokens:      2,
					RefillRate:     2,
					RefillInterval: 2 * time.Minute,
				},
				MongoDB: MongoDBConfig{
					Seed: false,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.configPath, tt.configEnv)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
