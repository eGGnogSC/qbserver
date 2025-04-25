// auth/models.go
package auth

import (
    "time"
)

// OAuthToken represents token data from QuickBooks
type OAuthToken struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    TokenType    string    `json:"token_type"`
    ExpiresIn    int       `json:"expires_in"`
    ExpiresAt    time.Time `json:"expires_at"`
    RealmID      string    `json:"realm_id"` // Company ID in QuickBooks
}

// TokenStore interface for different token storage implementations
type TokenStore interface {
    SaveToken(userID string, token *OAuthToken) error
    GetToken(userID string) (*OAuthToken, error)
    DeleteToken(userID string) error
}

// OAuthConfig holds OAuth 2.0 configuration
type OAuthConfig struct {
    ClientID     string
    ClientSecret string
    RedirectURI  string
    Scopes       []string
    AuthURL      string
    TokenURL     string
    APIBaseURL   string
}
