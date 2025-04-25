// auth/token_store_fallback.go
package auth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// FallbackTokenStore provides a resilient token store with local cache
type FallbackTokenStore struct {
	redisStore  *RedisTokenStore
	localCache  map[string]*OAuthToken
	cacheMutex  sync.RWMutex
	healthCheck func() bool
}

// NewFallbackTokenStore creates a token store with Redis and local fallback
func NewFallbackTokenStore(redisClient redis.UniversalClient, prefix string, healthCheck func() bool) *FallbackTokenStore {
	return &FallbackTokenStore{
		redisStore:  NewRedisTokenStore(redisClient, prefix),
		localCache:  make(map[string]*OAuthToken),
		healthCheck: healthCheck,
	}
}

// SaveToken stores a token in Redis and local cache
func (s *FallbackTokenStore) SaveToken(userID string, token *OAuthToken) error {
	// Update local cache
	s.cacheMutex.Lock()
	s.localCache[userID] = token
	s.cacheMutex.Unlock()
	
	// If Redis is healthy, update it too
	if s.healthCheck() {
		if err := s.redisStore.SaveToken(userID, token); err != nil {
			log.Printf("Warning: Failed to save token to Redis: %v", err)
			// Continue with just local cache
		}
	}
	
	return nil
}

// GetToken retrieves a token, trying Redis first, falling back to local cache
func (s *FallbackTokenStore) GetToken(userID string) (*OAuthToken, error) {
	// Try Redis first if healthy
	if s.healthCheck() {
		token, err := s.redisStore.GetToken(userID)
		if err == nil {
			// Update local cache
			s.cacheMutex.Lock()
			s.localCache[userID] = token
			s.cacheMutex.Unlock()
			return token, nil
		}
		// Redis failed, log and fall back to cache
		log.Printf("Warning: Failed to get token from Redis: %v", err)
	}
	
	// Try local cache
	s.cacheMutex.RLock()
	token, exists := s.localCache[userID]
	s.cacheMutex.RUnlock()
	
	if exists {
		return token, nil
	}
	
	return nil, fmt.Errorf("token not found for user")
}

// DeleteToken removes a token from both stores
func (s *FallbackTokenStore) DeleteToken(userID string) error {
	// Remove from local cache
	s.cacheMutex.Lock()
	delete(s.localCache, userID)
	s.cacheMutex.Unlock()
	
	// If Redis is healthy, remove from there too
	if s.healthCheck() {
		if err := s.redisStore.DeleteToken(userID); err != nil {
			log.Printf("Warning: Failed to delete token from Redis: %v", err)
			// Continue with just local removal
		}
	}
	
	return nil
}

// StartReplicationRoutine begins background sync of local cache to Redis
func (s *FallbackTokenStore) StartReplicationRoutine(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !s.healthCheck() {
					continue
				}
				
				// Copy tokens that need replication
				s.cacheMutex.RLock()
				tokensToReplicate := make(map[string]*OAuthToken)
				for id, token := range s.localCache {
					tokensToReplicate[id] = token
				}
				s.cacheMutex.RUnlock()
				
				// Replicate to Redis
				for id, token := range tokensToReplicate {
					if err := s.redisStore.SaveToken(id, token); err != nil {
						log.Printf("Replication error for user %s: %v", id, err)
					}
				}
			}
		}
	}()
}
