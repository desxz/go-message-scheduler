package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRateLimiter_Allow(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name                string
		config              RateLimiterConfig
		requests            int
		expectedAllowedReqs int
	}{
		{
			name: "allow up to max tokens then block",
			config: RateLimiterConfig{
				MaxTokens:      5,
				RefillRate:     2,
				RefillInterval: time.Minute,
			},
			requests:            10,
			expectedAllowedReqs: 5,
		},
		{
			name: "single token bucket",
			config: RateLimiterConfig{
				MaxTokens:      1,
				RefillRate:     1,
				RefillInterval: time.Minute,
			},
			requests:            3,
			expectedAllowedReqs: 1,
		},
		{
			name: "zero capacity bucket should block all requests",
			config: RateLimiterConfig{
				MaxTokens:      0,
				RefillRate:     0,
				RefillInterval: time.Minute,
			},
			requests:            5,
			expectedAllowedReqs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.config, logger)
			defer rl.Stop()

			allowed := 0
			for i := 0; i < tt.requests; i++ {
				if rl.Allow() {
					allowed++
				}
			}

			assert.Equal(t, tt.expectedAllowedReqs, allowed, "Number of allowed requests should match expected")
		})
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name               string
		config             RateLimiterConfig
		initialConsumption int
		waitRefills        int
		expectedTokens     int
	}{
		{
			name: "refill to max capacity",
			config: RateLimiterConfig{
				MaxTokens:      5,
				RefillRate:     2,
				RefillInterval: 10 * time.Millisecond,
			},
			initialConsumption: 5,
			waitRefills:        3,
			expectedTokens:     5,
		},
		{
			name: "partial refill",
			config: RateLimiterConfig{
				MaxTokens:      10,
				RefillRate:     3,
				RefillInterval: 10 * time.Millisecond,
			},
			initialConsumption: 6,
			waitRefills:        2,
			expectedTokens:     10,
		},
		{
			name: "no refill needed",
			config: RateLimiterConfig{
				MaxTokens:      5,
				RefillRate:     2,
				RefillInterval: 10 * time.Millisecond,
			},
			initialConsumption: 0,
			waitRefills:        3,
			expectedTokens:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.config, logger)
			defer rl.Stop()

			for i := 0; i < tt.initialConsumption; i++ {
				rl.Allow()
			}

			time.Sleep(tt.config.RefillInterval * time.Duration(tt.waitRefills+1))

			remaining := 0
			for i := 0; i < tt.config.MaxTokens*2; i++ {
				if rl.Allow() {
					remaining++
				} else {
					break
				}
			}

			assert.Equal(t, tt.expectedTokens, remaining, "Number of tokens after refill should match expected")
		})
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	logger, _ := zap.NewDevelopment()

	config := RateLimiterConfig{
		MaxTokens:      100,
		RefillRate:     10,
		RefillInterval: 50 * time.Millisecond,
	}

	rl := NewRateLimiter(config, logger)
	defer rl.Stop()

	concurrency := 10
	requestsPerRoutine := 20

	results := make(chan bool, concurrency*requestsPerRoutine)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < requestsPerRoutine; j++ {
				results <- rl.Allow()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	allowed := 0
	totalRequests := concurrency * requestsPerRoutine
	for i := 0; i < totalRequests; i++ {
		if <-results {
			allowed++
		}
	}

	assert.LessOrEqual(t, allowed, config.MaxTokens+30, "Allowed requests should not significantly exceed max tokens")
	assert.GreaterOrEqual(t, allowed, config.MaxTokens, "Should allow at least maxTokens requests")
}

func TestRateLimiter_Stop(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := RateLimiterConfig{
		MaxTokens:      5,
		RefillRate:     1,
		RefillInterval: 10 * time.Millisecond,
	}

	rl := NewRateLimiter(config, logger)

	rl.Stop()

	assert.NotPanics(t, func() {
		rl.Stop()
	})

	for i := 0; i < config.MaxTokens; i++ {
		assert.True(t, rl.Allow())
	}

	assert.False(t, rl.Allow())

	time.Sleep(50 * time.Millisecond)

	assert.False(t, rl.Allow())
}
