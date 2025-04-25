// auth/middleware.go
package auth

import (
    "context"
    "errors"
    "net/http"
)

// contextKey is a custom type for context keys
type contextKey string

// Context keys
const (
    UserIDKey   contextKey = "user_id"
    TokenKey    contextKey = "token"
    CompanyIDKey contextKey = "company_id"
)

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
    userID, _ := ctx.Value(UserIDKey).(string)
    return userID
}

// GetToken extracts token from context
func GetToken(ctx context.Context) *OAuthToken {
    token, _ := ctx.Value(TokenKey).(*OAuthToken)
    return token
}

// GetCompanyID extracts company ID from context
func GetCompanyID(ctx context.Context) (string, error) {
    companyID, ok := ctx.Value(CompanyIDKey).(string)
    if !ok || companyID == "" {
        return "", errors.New("company ID not found in context")
    }
    return companyID, nil
}

// UserMiddleware sets user ID in the request context
// Replace this with your actual user authentication logic
func UserMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Example: Get user ID from Authorization header or session
        // In a real app, you'd validate JWT, session token, etc.
        userID := r.Header.Get("X-User-ID")
        if userID == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Set user ID in context
        ctx := context.WithValue(r.Context(), UserIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// QBAuthMiddleware ensures the request has a valid QuickBooks token
func QBAuthMiddleware(service *Service) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Get user ID from context
            userID := GetUserID(r.Context())
            if userID == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // Get and validate token
            token, err := service.GetValidToken(r.Context(), userID)
            if err != nil {
                http.Error(w, "QuickBooks authentication required", http.StatusUnauthorized)
                return
            }
            
            // Ensure company ID exists
            if token.RealmID == "" {
                http.Error(w, "QuickBooks company not connected", http.StatusUnauthorized)
                return
            }
            
            // Set token and company ID in context
            ctx := context.WithValue(r.Context(), TokenKey, token)
            ctx = context.WithValue(ctx, CompanyIDKey, token.RealmID)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
