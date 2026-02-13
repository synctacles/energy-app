// Command energy-api is the Synctacles Energy REST API server.
// It provides energy price data, best window calculations, and GO/WAIT/AVOID
// recommendations via a RESTful HTTP API.
//
// This replaces the Python FastAPI implementation with a high-performance
// Go service using Chi router and pgx for PostgreSQL.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/synctacles/energy-go/internal/api/handlers"
	apimiddleware "github.com/synctacles/energy-go/internal/api/middleware"
)

const (
	version        = "2.0.0"
	defaultPort    = "8002"
	defaultDBURL   = "postgres://synctacles_dev@localhost:5432/energy_dev?sslmode=disable"
	defaultAuthURL = "http://localhost:8000"
)

func main() {
	// Load .env file first (optional)
	_ = godotenv.Load()

	// Setup structured logging (file or stdout)
	logFile := getEnv("LOG_FILE", "")
	logLevel := getEnv("LOG_LEVEL", "info")

	var logOutput *os.File
	if logFile != "" {
		var err error
		logOutput, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", logFile, err)
			os.Exit(1)
		}
		defer logOutput.Close()
	} else {
		logOutput = os.Stdout
	}

	// Parse log level
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(logOutput, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	// Configuration from environment
	port := getEnv("PORT", defaultPort)
	dbURL := getEnv("DATABASE_URL", defaultDBURL)
	authURL := getEnv("AUTH_SERVICE_URL", defaultAuthURL)

	logger.Info("starting energy API server",
		"version", version,
		"port", port,
		"auth_url", authURL,
	)

	// Database connection pool
	dbpool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	// Verify database connection
	if err := dbpool.Ping(context.Background()); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	logger.Info("database connected")

	// Initialize handlers
	h := handlers.New(dbpool, logger)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimiddleware.StructuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS (adjust origins as needed)
	r.Use(middleware.SetHeader("Access-Control-Allow-Origin", "*"))
	r.Use(middleware.SetHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS"))
	r.Use(middleware.SetHeader("Access-Control-Allow-Headers", "Content-Type, X-API-Key"))

	// Health check (no auth required)
	r.Get("/health", h.Health)

	// API routes (with authentication)
	r.Route("/api/v1", func(r chi.Router) {
		// Auth middleware for all /api/v1/* routes
		r.Use(apimiddleware.CentralAuth(authURL, "energy", logger))

		// Price endpoints
		r.Get("/prices", h.GetPrices)           // Free tier
		r.Get("/now", h.GetNow)                 // Free tier (deprecated)

		// Pro tier endpoints
		r.Get("/dashboard", h.GetDashboard)     // Pro
		r.Get("/best-window", h.GetBestWindow)  // Pro
		r.Get("/energy-action", h.GetAction)    // Pro
		r.Get("/tomorrow", h.GetTomorrow)       // Pro
		r.Get("/balance", h.GetBalance)         // Pro
	})

	// Metrics endpoint (no auth)
	r.Get("/metrics", h.Metrics)

	// Start server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
