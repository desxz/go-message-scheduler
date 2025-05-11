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
		maxTokens           int
		refillRate          int
		refillInterval      time.Duration
		requests            int
		expectedAllowedReqs int
	}{
		{
			name:                "allow up to max tokens then block",
			maxTokens:           5,
			refillRate:          2,
			refillInterval:      time.Minute,
			requests:            10,
			expectedAllowedReqs: 5,
		},
		{
			name:                "single token bucket",
			maxTokens:           1,
			refillRate:          1,
			refillInterval:      time.Minute,
			requests:            3,
			expectedAllowedReqs: 1,
		},
		{
			name:                "zero capacity bucket should block all requests",
			maxTokens:           0,
			refillRate:          0,
			refillInterval:      time.Minute,
			requests:            5,
			expectedAllowedReqs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.maxTokens, tt.refillRate, tt.refillInterval, logger)
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
		maxTokens          int
		refillRate         int
		refillInterval     time.Duration
		initialConsumption int
		waitRefills        int
		expectedTokens     int
	}{
		{
			name:               "refill to max capacity",
			maxTokens:          5,
			refillRate:         2,
			refillInterval:     10 * time.Millisecond,
			initialConsumption: 5,
			waitRefills:        3,
			expectedTokens:     5,
		},
		{
			name:               "partial refill",
			maxTokens:          10,
			refillRate:         3,
			refillInterval:     10 * time.Millisecond,
			initialConsumption: 6,
			waitRefills:        2,
			expectedTokens:     10,
		},
		{
			name:               "no refill needed",
			maxTokens:          5,
			refillRate:         2,
			refillInterval:     10 * time.Millisecond,
			initialConsumption: 0,
			waitRefills:        3,
			expectedTokens:     5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.maxTokens, tt.refillRate, tt.refillInterval, logger)
			defer rl.Stop()

			for i := 0; i < tt.initialConsumption; i++ {
				rl.Allow()
			}

			time.Sleep(tt.refillInterval * time.Duration(tt.waitRefills+1))

			remaining := 0
			for i := 0; i < tt.maxTokens*2; i++ {
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

	maxTokens := 100
	rl := NewRateLimiter(maxTokens, 10, 50*time.Millisecond, logger)
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

	assert.LessOrEqual(t, allowed, maxTokens+30, "Allowed requests should not significantly exceed max tokens")
	assert.GreaterOrEqual(t, allowed, maxTokens, "Should allow at least maxTokens requests")
}

func TestRateLimiter_Stop(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	rl := NewRateLimiter(5, 1, 10*time.Millisecond, logger)

	rl.Stop()

	assert.NotPanics(t, func() {
		rl.Stop()
	})

	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow())
	}

	assert.False(t, rl.Allow())

	time.Sleep(50 * time.Millisecond)

	assert.False(t, rl.Allow())
}
