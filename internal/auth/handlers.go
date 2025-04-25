// auth/handlers.go
package auth

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "net/http"
    "time"
)

// Handler provides HTTP handlers for auth flows
type Handler struct {
    service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
    return &Handler{
        service: service,
    }
}

// generateState creates a secure random state for OAuth
func (h *Handler) generateState() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}

// ConnectHandler initiates the QuickBooks authorization flow
func (h *Handler) ConnectHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from session or auth
    userID := GetUserID(r.Context())
    if userID == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Generate state parameter
    state, err := h.generateState()
    if err != nil {
        http.Error(w, "Failed to generate state", http.StatusInternalServerError)
        return
    }
    
    // Save state in session for verification
    session := GetSession(r)
    session.Values["qb_state"] = state
    session.Values["qb_state_expiry"] = time.Now().Add(10 * time.Minute).Unix()
    if err := session.Save(r, w); err != nil {
        http.Error(w, "Failed to save session", http.StatusInternalServerError)
        return
    }
    
    // Redirect to QuickBooks authorization page
    authURL := h.service.GetAuthorizationURL(state)
    http.Redirect(w, r, authURL, http.StatusFound)
}

// CallbackHandler handles the OAuth callback from QuickBooks
func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from session or auth
    userID := GetUserID(r.Context())
    if userID == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Get query parameters
    query := r.URL.Query()
    code := query.Get("code")
    state := query.Get("state")
    realmID := query.Get("realmId")
    
    if code == "" || state == "" {
        http.Error(w, "Invalid callback parameters", http.StatusBadRequest)
        return
    }
    
    // Verify state parameter
    session := GetSession(r)
    savedState, ok := session.Values["qb_state"].(string)
    if !ok || savedState != state {
        http.Error(w, "Invalid state parameter", http.StatusBadRequest)
        return
    }
    
    // Verify state hasn't expired
    expiry, ok := session.Values["qb_state_expiry"].(int64)
    if !ok || time.Now().Unix() > expiry {
        http.Error(w, "State parameter expired", http.StatusBadRequest)
        return
    }
    
    // Clean up session
    delete(session.Values, "qb_state")
    delete(session.Values, "qb_state_expiry")
    if err := session.Save(r, w); err != nil {
        http.Error(w, "Failed to save session", http.StatusInternalServerError)
        return
    }
    
    // Exchange code for token
    token, err := h.service.HandleCallback(r.Context(), code, state, userID)
    if err != nil {
        http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Set realm ID from callback
    if realmID != "" {
        token.RealmID = realmID
        // Save updated token
        if err := h.service.tokenStore.SaveToken(userID, token); err != nil {
            http.Error(w, "Failed to save token with realm ID", http.StatusInternalServerError)
            return
        }
    }
    
    // Return success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status":   "success",
        "realm_id": token.RealmID,
    })
}

// DisconnectHandler revokes QuickBooks tokens
func (h *Handler) DisconnectHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from session or auth
    userID := GetUserID(r.Context())
    if userID == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Disconnect from QuickBooks
    if err := h.service.Disconnect(r.Context(), userID); err != nil {
        http.Error(w, "Failed to disconnect: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return success response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
    })
}

// StatusHandler returns the connection status
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
    // Get user ID from session or auth
    userID := GetUserID(r.Context())
    if userID == "" {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Check if user has a token
    token, err := h.service.tokenStore.GetToken(userID)
    if err != nil {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "connected": false,
        })
        return
    }
    
    // Return connection status
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "connected": true,
        "realm_id":  token.RealmID,
        "expires_at": token.ExpiresAt,
    })
}
