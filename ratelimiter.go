package main

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

type RateLimiterConfig struct {
	MaxTokens      int           `mapstructure:"maxTokens"`
	RefillRate     int           `mapstructure:"refillRate"`
	RefillInterval time.Duration `mapstructure:"refillInterval"`
	Enabled        bool          `mapstructure:"enabled"`
}

type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillRate     int
	refillInterval time.Duration
	mu             sync.Mutex
	logger         *zap.Logger
	stopRefill     chan struct{}
	stopOnce       sync.Once
}

func NewRateLimiter(config RateLimiterConfig, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		tokens:         config.MaxTokens,
		maxTokens:      config.MaxTokens,
		refillRate:     config.RefillRate,
		refillInterval: config.RefillInterval,
		logger:         logger.With(zap.String("component", "ratelimiter")),
		stopRefill:     make(chan struct{}),
	}

	go rl.startRefill()

	return rl
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens > 0 {
		rl.tokens--
		rl.logger.Debug("Token consumed", zap.Int("remaining", rl.tokens))
		return true
	}

	rl.logger.Debug("Rate limit exceeded, no tokens available")
	return false
}

func (rl *RateLimiter) startRefill() {
	ticker := time.NewTicker(rl.refillInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.refill()
		case <-rl.stopRefill:
			return
		}
	}
}

func (rl *RateLimiter) refill() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.tokens = min(rl.tokens+rl.refillRate, rl.maxTokens)
	rl.logger.Debug("Tokens refilled", zap.Int("current", rl.tokens), zap.Int("refillRate", rl.refillRate))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopRefill)
	})
}
