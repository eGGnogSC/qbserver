// routes/routes.go
package routes

import (
	"github.com/gorilla/mux"
	"github.com/eGGnogSC/qbserver/internal/auth"
	"github.com/eGGnogSC/qbserver/internal/invoice"
	"github.com/eGGnogSC/qbserver/internal/customer"
	"github.com/eGGnogSC/qbserver/internal/item"
	"github.com/eGGnogSC/qbserver/internal/payment"
	"github.com/eGGnogSC/qbserver/nlp"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	router *mux.Router,
	authHandler *auth.Handler,
	authService *auth.Service,
	invoiceHandler *invoice.Handler,
	customerHandler *customer.Handler,
	itemHandler *item.Handler,
	paymentHandler *payment.Handler,
	agentHandler *nlp.AgentHandler,
) {
	// Register auth routes
	RegisterAuthRoutes(router, authHandler)
	
	// API routes - protected with QuickBooks auth
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.Use(auth.UserMiddleware)
	apiRouter.Use(auth.QBAuthMiddleware(authService))
	
	// Register domain-specific routes
	RegisterInvoiceRoutes(apiRouter, invoiceHandler)
	RegisterCustomerRoutes(apiRouter, customerHandler)
	RegisterItemRoutes(apiRouter, itemHandler)
	RegisterPaymentRoutes(apiRouter, paymentHandler)
	
	// Register NLP agent routes
	agentRouter := router.PathPrefix("/agent").Subrouter()
	agentRouter.Use(auth.UserMiddleware)
	agentRouter.HandleFunc("/query", agentHandler.ProcessCommand).Methods("POST")
}
