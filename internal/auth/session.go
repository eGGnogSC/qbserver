// auth/session.go
package auth

import (
    "net/http"
    
    "github.com/gorilla/sessions"
)

var (
    store *sessions.CookieStore
)

// InitSessionStore initializes the session store
func InitSessionStore(secret []byte) {
    store = sessions.NewCookieStore(secret)
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   86400 * 30, // 30 days
        HttpOnly: true,
        Secure:   true, // Require HTTPS
        SameSite: http.SameSiteStrictMode,
    }
}

// GetSession retrieves the session
func GetSession(r *http.Request) *sessions.Session {
    session, _ := store.Get(r, "qb-auth-session")
    return session
}
