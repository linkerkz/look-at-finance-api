package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/lukivan8/look-at-finance-api/internal/modules/auth"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
	"github.com/lukivan8/look-at-finance-api/internal/shared/middleware"
)

func main() {
	// Load .env file (ignore error if not found - use system env vars)
	_ = godotenv.Load()

	// Initialize logger
	log := logger.New()
	defer log.Sync()

	// Load configuration from environment
	cfg := loadConfig()

	// Initialize database connection
	db, err := database.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to database", err)
	}
	defer db.Close()

	// Initialize router
	r := chi.NewRouter()

	// Global middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(chimw.Timeout(60 * time.Second))

	// Initialize modules
	authModule := auth.NewModule(db, log, cfg.JWTSecret)
	storesModule := stores.NewModule(db, log)
	productsModule := products.NewModule(db, log)

	// Receipts module needs store and product services for dependency injection
	receiptsModule := receipts.NewModule(db, log, storesModule.Service, productsModule.Service)
	expensesModule := expenses.NewModule(db, log)

	// Auth middleware for protected routes
	authMiddleware := middleware.Auth(cfg.JWTSecret)

	// Register routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			authModule.RegisterPublicRoutes(r)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			authModule.RegisterProtectedRoutes(r)
			storesModule.RegisterRoutes(r)
			productsModule.RegisterRoutes(r)
			receiptsModule.RegisterRoutes(r)
			expensesModule.RegisterRoutes(r)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Info(fmt.Sprintf("server starting on port %s", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed to start", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", err)
	}

	log.Info("server exited properly")
}

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "3000"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/finance?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
