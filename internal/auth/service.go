// auth/service.go
package auth

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// Service handles OAuth 2.0 operations
type Service struct {
    config     OAuthConfig
    tokenStore TokenStore
}

// NewService creates a new auth service
func NewService(config OAuthConfig, tokenStore TokenStore) *Service {
    return &Service{
        config:     config,
        tokenStore: tokenStore,
    }
}

// GetAuthorizationURL generates the QuickBooks authorization URL
func (s *Service) GetAuthorizationURL(state string) string {
    u, _ := url.Parse(s.config.AuthURL)
    q := u.Query()
    
    q.Set("client_id", s.config.ClientID)
    q.Set("response_type", "code")
    q.Set("scope", strings.Join(s.config.Scopes, " "))
    q.Set("redirect_uri", s.config.RedirectURI)
    q.Set("state", state)
    
    u.RawQuery = q.Encode()
    return u.String()
}

// HandleCallback processes the OAuth callback and exchanges the code for tokens
func (s *Service) HandleCallback(ctx context.Context, code, state, userID string) (*OAuthToken, error) {
    // Prepare token exchange request
    data := url.Values{}
    data.Set("grant_type", "authorization_code")
    data.Set("code", code)
    data.Set("redirect_uri", s.config.RedirectURI)
    
    // Execute token exchange
    token, err := s.executeTokenRequest(ctx, data)
    if err != nil {
        return nil, err
    }
    
    // Set expiry time
    token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
    
    // Save token
    if err := s.tokenStore.SaveToken(userID, token); err != nil {
        return nil, fmt.Errorf("failed to save token: %w", err)
    }
    
    return token, nil
}

// RefreshToken refreshes an expired access token
func (s *Service) RefreshToken(ctx context.Context, userID string) (*OAuthToken, error) {
    // Get current token
    token, err := s.tokenStore.GetToken(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get token for refresh: %w", err)
    }
    
    // Prepare refresh request
    data := url.Values{}
    data.Set("grant_type", "refresh_token")
    data.Set("refresh_token", token.RefreshToken)
    
    // Execute refresh
    newToken, err := s.executeTokenRequest(ctx, data)
    if err != nil {
        return nil, err
    }
    
    // Update token fields
    newToken.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
    newToken.RealmID = token.RealmID // Preserve realm ID
    
    // If the refresh token was not returned, reuse the existing one
    if newToken.RefreshToken == "" {
        newToken.RefreshToken = token.RefreshToken
    }
    
    // Save updated token
    if err := s.tokenStore.SaveToken(userID, newToken); err != nil {
        return nil, fmt.Errorf("failed to save refreshed token: %w", err)
    }
    
    return newToken, nil
}

// executeTokenRequest performs the actual token request to QuickBooks
func (s *Service) executeTokenRequest(ctx context.Context, data url.Values) (*OAuthToken, error) {
    req, err := http.NewRequestWithContext(ctx, "POST", s.config.TokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, fmt.Errorf("failed to create token request: %w", err)
    }
    
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Add("Accept", "application/json")
    req.SetBasicAuth(s.config.ClientID, s.config.ClientSecret)
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("token request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, body)
    }
    
    var token OAuthToken
    if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
        return nil, fmt.Errorf("failed to parse token response: %w", err)
    }
    
    // Parse realm ID from query params if it exists
    if realmID := req.URL.Query().Get("realm_id"); realmID != "" {
        token.RealmID = realmID
    }
    
    return &token, nil
}

// GetValidToken returns a valid token, refreshing it if necessary
func (s *Service) GetValidToken(ctx context.Context, userID string) (*OAuthToken, error) {
    token, err := s.tokenStore.GetToken(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get token: %w", err)
    }
    
    // Check if token is expired or about to expire (within 5 minutes)
    if time.Until(token.ExpiresAt) < 5*time.Minute {
        token, err = s.RefreshToken(ctx, userID)
        if err != nil {
            return nil, fmt.Errorf("failed to refresh token: %w", err)
        }
    }
    
    return token, nil
}

// Disconnect revokes tokens and removes from storage
func (s *Service) Disconnect(ctx context.Context, userID string) error {
    // Get token
    token, err := s.tokenStore.GetToken(userID)
    if err != nil {
        return fmt.Errorf("failed to get token for revocation: %w", err)
    }
    
    // Revoke access token
    if err := s.revokeToken(ctx, token.AccessToken); err != nil {
        return err
    }
    
    // Revoke refresh token
    if err := s.revokeToken(ctx, token.RefreshToken); err != nil {
        return err
    }
    
    // Remove from storage
    return s.tokenStore.DeleteToken(userID)
}

// revokeToken revokes a token with QuickBooks
func (s *Service) revokeToken(ctx context.Context, token string) error {
    data := url.Values{}
    data.Set("token", token)
    
    req, err := http.NewRequestWithContext(ctx, "POST", s.config.APIBaseURL+"/oauth2/v1/tokens/revoke", strings.NewReader(data.Encode()))
    if err != nil {
        return fmt.Errorf("failed to create revoke request: %w", err)
    }
    
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    req.SetBasicAuth(s.config.ClientID, s.config.ClientSecret)
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("revoke request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("revoke request failed with status %d: %s", resp.StatusCode, body)
    }
    
    return nil
}
