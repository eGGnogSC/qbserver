// infrastructure/redis/client.go
package redis

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/go-redis/redis/v8"
)

// Config holds Redis connection configuration
type Config struct {
	Addresses          []string
	Password           string
	DB                 int
	MaxRetries         int
	MinRetryBackoff    time.Duration
	MaxRetryBackoff    time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PoolSize           int
	MinIdleConns       int
	MaxConnAge         time.Duration
	PoolTimeout        time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	EnableTLS          bool
}

// DefaultConfig returns a production-ready Redis configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:         3,
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    512 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        3 * time.Second,
		WriteTimeout:       3 * time.Second,
		PoolSize:           10, // Adjust based on expected concurrent connections
		MinIdleConns:       2,
		MaxConnAge:         30 * time.Minute,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
	}
}

// NewClient creates a new Redis client with optimized connection pooling
func NewClient(cfg Config) *redis.Client {
	options := &redis.Options{
		Addr:               cfg.Addresses[0], // Use first address for single instance
		Password:           cfg.Password,
		DB:                 cfg.DB,
		MaxRetries:         cfg.MaxRetries,
		MinRetryBackoff:    cfg.MinRetryBackoff,
		MaxRetryBackoff:    cfg.MaxRetryBackoff,
		DialTimeout:        cfg.DialTimeout,
		ReadTimeout:        cfg.ReadTimeout,
		WriteTimeout:       cfg.WriteTimeout,
		PoolSize:           cfg.PoolSize,
		MinIdleConns:       cfg.MinIdleConns,
		MaxConnAge:         cfg.MaxConnAge,
		PoolTimeout:        cfg.PoolTimeout,
		IdleTimeout:        cfg.IdleTimeout,
		IdleCheckFrequency: cfg.IdleCheckFrequency,
	}

	if cfg.EnableTLS {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewClient(options)
}

// NewClusterClient creates a Redis cluster client for high availability
func NewClusterClient(cfg Config) *redis.ClusterClient {
	options := &redis.ClusterOptions{
		Addrs:              cfg.Addresses,
		Password:           cfg.Password,
		MaxRetries:         cfg.MaxRetries,
		MinRetryBackoff:    cfg.MinRetryBackoff,
		MaxRetryBackoff:    cfg.MaxRetryBackoff,
		DialTimeout:        cfg.DialTimeout,
		ReadTimeout:        cfg.ReadTimeout,
		WriteTimeout:       cfg.WriteTimeout,
		PoolSize:           cfg.PoolSize,
		MinIdleConns:       cfg.MinIdleConns,
		MaxConnAge:         cfg.MaxConnAge,
		PoolTimeout:        cfg.PoolTimeout,
		IdleTimeout:        cfg.IdleTimeout,
		IdleCheckFrequency: cfg.IdleCheckFrequency,
	}

	if cfg.EnableTLS {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewClusterClient(options)
}
