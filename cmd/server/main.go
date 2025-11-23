// Package main provides the entry point for the HTTP server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/festy23/avito_internship/internal/config"
	"github.com/festy23/avito_internship/internal/database/database"
	"github.com/festy23/avito_internship/internal/database/migrate"
	"github.com/festy23/avito_internship/internal/health"
	"github.com/festy23/avito_internship/internal/middleware"
	pullrequestRouter "github.com/festy23/avito_internship/internal/pullrequest/router"
	teamRouter "github.com/festy23/avito_internship/internal/team/router"
	userRouter "github.com/festy23/avito_internship/internal/user/router"
	"github.com/festy23/avito_internship/pkg/logger"
)

func main() {
	// Load application configuration
	appConfig := config.LoadFromEnv()

	// Validate configuration
	if err := appConfig.Validate(); err != nil {
		panic(fmt.Sprintf("invalid configuration: %v", err))
	}

	// Set Gin mode
	gin.SetMode(appConfig.GinMode)

	// Initialize logger
	log, err := logger.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			_ = syncErr
		}
	}()

	// Initialize database
	db, err := database.New()
	if err != nil {
		log.Fatalw("failed to connect to database", "error", err)
	}

	// Apply database migrations
	if err := migrate.Migrate(db); err != nil {
		log.Fatalw("failed to run migrations", "error", err)
	}

	// Setup router
	r := gin.New()

	// Apply middleware (order matters: recovery first, then logger)
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Logger(log))

	// Register health check endpoint
	healthHandler := health.New(db, log)
	r.GET("/health", healthHandler.Check)

	teamRouter.RegisterRoutes(r, db, log)
	userRouter.RegisterRoutes(r, db, log)
	pullrequestRouter.RegisterRoutes(r, db, log)

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         appConfig.Server.GetAddress(),
		Handler:      r,
		ReadTimeout:  appConfig.Server.ReadTimeout,
		WriteTimeout: appConfig.Server.WriteTimeout,
		IdleTimeout:  appConfig.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Infow("starting server", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalw("failed to start server", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infow("shutting down server")

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorw("server forced to shutdown", "error", err)
	} else {
		log.Infow("HTTP server stopped")
	}

	// Close database connection
	if err := database.Close(db); err != nil {
		log.Errorw("failed to close database", "error", err)
	} else {
		log.Infow("database connection closed")
	}

	log.Infow("server exited")
}
