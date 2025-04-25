// infrastructure/container.go
package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/eGGnogSC/qbserver/config"
	"github.com/eGGnogSC/qbserver/internal/auth"
	"github.com/eGGnogSC/qbserver/internal/customer"
	"github.com/eGGnogSC/qbserver/internal/invoice"
	"github.com/eGGnogSC/qbserver/internal/item"
	"github.com/eGGnogSC/qbserver/internal/payment"
	"github.com/eGGnogSC/qbserver/nlp"
	"github.com/eGGnogSC/qbserver/pkg/qbclient"
)

// Container provides application dependencies
type Container struct {
	// Services
	AuthService     *auth.Service
	InvoiceService  *invoice.Service
	CustomerService *customer.Service
	ItemService     *item.Service
	PaymentService  *payment.Service
	
	// Handlers
	AuthHandler     *auth.Handler
	InvoiceHandler  *invoice.Handler
	CustomerHandler *customer.Handler
	ItemHandler     *item.Handler
	PaymentHandler  *payment.Handler
	AgentHandler    *nlp.AgentHandler
	
	// Infrastructure
	RedisClient     redis.UniversalClient
	RedisHealth     *redis.HealthChecker
	TokenStore      auth.TokenStore
	QBClient        *qbclient.Client
}

// NewContainer creates and initializes the dependency container
func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	container := &Container{}
	
	// Initialize Redis client based on configuration
	var redisClient redis.UniversalClient

	if len(config.Redis.Addresses) > 1 {
		// Use cluster client for multiple nodes
		redisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      config.Redis.Addresses,
			Password:   config.Redis.Password,
			// Other options from config
		})
	} else {
		// Use single node client
		redisClient = redis.NewClient(&redis.Options{
			Addr:       config.Redis.Addresses[0],
			Password:   config.Redis.Password,
			DB:         config.Redis.DB,
			// Other options from config
		})
	}

	// Create health checker
	redisHealth := redis.NewHealthChecker(redisClient, 30*time.Second)

	// Create token store with Redis
	tokenStore := auth.NewRedisTokenStore(redisClient, config.Redis.KeyPrefix)

	// Initialize services
	container.AuthService = auth.NewService(auth.OAuthConfig{
		ClientID:     cfg.QuickBooks.ClientID,
		ClientSecret: cfg.QuickBooks.ClientSecret,
		RedirectURI:  cfg.QuickBooks.RedirectURI,
		Scopes:       cfg.QuickBooks.Scopes,
		AuthURL:      cfg.QuickBooks.AuthURL,
		TokenURL:     cfg.QuickBooks.TokenURL,
		APIBaseURL:   cfg.QuickBooks.APIBaseURL,
	}, container.TokenStore)
	
	// Initialize QuickBooks client
	container.QBClient = qbclient.NewClient(
		cfg.QuickBooks.APIBaseURL,
		cfg.QuickBooks.ClientID,
		cfg.QuickBooks.ClientSecret,
		container.AuthService,
	)
	
	// Initialize domain services
	container.CustomerService = customer.NewService(container.QBClient)
	container.ItemService = item.NewService(container.QBClient)
	container.InvoiceService = invoice.NewService(
		container.QBClient, 
		container.CustomerService, 
		container.ItemService,
	)
	container.PaymentService = payment.NewService(container.QBClient)
	
	// Initialize handlers
	container.AuthHandler = auth.NewHandler(container.AuthService)
	container.CustomerHandler = customer.NewHandler(container.CustomerService)
	container.ItemHandler = item.NewHandler(container.ItemService)
	container.InvoiceHandler = invoice.NewHandler(container.InvoiceService)
	container.PaymentHandler = payment.NewHandler(container.PaymentService)
	
	// Initialize NLP processors
	invoiceProcessor := nlp.NewInvoiceProcessor(
		container.CustomerService,
		container.ItemService,
		container.InvoiceService,
	)
	
	// Initialize Agent handler
	container.AgentHandler = nlp.NewAgentHandler(invoiceProcessor)
	
	return container, nil
}

// Shutdown gracefully closes connections
func (c *Container) Shutdown() {
	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		}
	}
}
