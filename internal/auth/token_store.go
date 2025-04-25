// auth/token_store.go
package auth

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
)

// RedisTokenStore implements TokenStore using Redis
type RedisTokenStore struct {
    client *redis.Client
    prefix string
}

// NewRedisTokenStore creates a new Redis-backed token store
func NewRedisTokenStore(client *redis.Client, prefix string) *RedisTokenStore {
    return &RedisTokenStore{
        client: client,
        prefix: prefix,
    }
}

// key generates the Redis key for a user's token
func (s *RedisTokenStore) key(userID string) string {
    return fmt.Sprintf("%s:token:%s", s.prefix, userID)
}

// SaveToken stores a token for a user
func (s *RedisTokenStore) SaveToken(userID string, token *OAuthToken) error {
    data, err := json.Marshal(token)
    if err != nil {
        return fmt.Errorf("failed to marshal token: %w", err)
    }
    
    // Calculate TTL based on token expiry plus a buffer
    ttl := time.Until(token.ExpiresAt) + (24 * time.Hour)
    
    err = s.client.Set(context.Background(), s.key(userID), data, ttl).Err()
    if err != nil {
        return fmt.Errorf("failed to save token: %w", err)
    }
    
    return nil
}

// GetToken retrieves a token for a user
func (s *RedisTokenStore) GetToken(userID string) (*OAuthToken, error) {
    data, err := s.client.Get(context.Background(), s.key(userID)).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, fmt.Errorf("no token found for user")
        }
        return nil, fmt.Errorf("failed to get token: %w", err)
    }
    
    var token OAuthToken
    if err := json.Unmarshal(data, &token); err != nil {
        return nil, fmt.Errorf("failed to unmarshal token: %w", err)
    }
    
    return &token, nil
}

// DeleteToken removes a user's token
func (s *RedisTokenStore) DeleteToken(userID string) error {
    err := s.client.Del(context.Background(), s.key(userID)).Err()
    if err != nil {
        return fmt.Errorf("failed to delete token: %w", err)
    }
    
    return nil
}
