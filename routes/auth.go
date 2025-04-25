// routes/auth.go
package routes

import (
	"github.com/gorilla/mux"
	"github.com/eGGnogSC/qbserver/internal/auth"
)

// RegisterAuthRoutes registers all authentication-related routes
func RegisterAuthRoutes(router *mux.Router, authHandler *auth.Handler) {
	// Public auth routes
	router.HandleFunc("/auth/connect", authHandler.ConnectHandler).Methods("GET")
	router.HandleFunc("/auth/callback", authHandler.CallbackHandler).Methods("GET")
	
	// Protected auth routes - require user authentication
	protectedRouter := router.PathPrefix("/auth").Subrouter()
	protectedRouter.Use(auth.UserMiddleware)
	protectedRouter.HandleFunc("/disconnect", authHandler.DisconnectHandler).Methods("POST")
	protectedRouter.HandleFunc("/status", authHandler.StatusHandler).Methods("GET")
}
