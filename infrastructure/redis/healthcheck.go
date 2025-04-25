// infrastructure/redis/healthcheck.go
package redis

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sony/gobreaker"
)

// HealthChecker monitors Redis connection health
type HealthChecker struct {
	client        redis.UniversalClient
	circuitBreaker *gobreaker.CircuitBreaker
	status        bool
	mu            sync.RWMutex
	checkInterval time.Duration
}

// NewHealthChecker creates a new Redis health checker
func NewHealthChecker(client redis.UniversalClient, checkInterval time.Duration) *HealthChecker {
	settings := gobreaker.Settings{
		Name:          "redis-circuit-breaker",
		MaxRequests:   0,
		Interval:      0,
		Timeout:       30 * time.Second,
		ReadyToTrip:   func(counts gobreaker.Counts) bool { return counts.ConsecutiveFailures >= 3 },
		OnStateChange: nil,
	}

	checker := &HealthChecker{
		client:        client,
		circuitBreaker: gobreaker.NewCircuitBreaker(settings),
		status:        false,
		checkInterval: checkInterval,
	}

	// Start periodic health checks
	go checker.startPeriodicChecks()

	return checker
}

// IsHealthy returns current Redis connection health status
func (h *HealthChecker) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

// Check performs a health check and returns the result
func (h *HealthChecker) Check(ctx context.Context) bool {
	result, err := h.circuitBreaker.Execute(func() (interface{}, error) {
		// Simple PING command to check connectivity
		return h.client.Ping(ctx).Result()
	})

	isHealthy := err == nil && result.(string) == "PONG"
	
	h.mu.Lock()
	h.status = isHealthy
	h.mu.Unlock()
	
	return isHealthy
}

// startPeriodicChecks begins regular health checking
func (h *HealthChecker) startPeriodicChecks() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		h.Check(ctx)
		cancel()
	}
}
