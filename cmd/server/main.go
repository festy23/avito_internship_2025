// Package main provides the entry point for the HTTP server.
package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/festy23/avito_internship/internal/database"
	"github.com/festy23/avito_internship/pkg/logger"
	pullrequestRouter "github.com/festy23/avito_internship/internal/pullrequest/router"
	teamRouter "github.com/festy23/avito_internship/internal/team/router"
	userRouter "github.com/festy23/avito_internship/internal/user/router"
)

// getEnv reads an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log, err := logger.New()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer func() {
		if syncErr := log.Sync(); syncErr != nil {
			// Ignore sync errors on shutdown
			_ = syncErr
		}
	}()

	db, err := database.New()
	if err != nil {
		log.Fatalw("failed to connect to database", "error", err)
	}

	r := gin.Default()

	teamRouter.RegisterRoutes(r, db, log)
	userRouter.RegisterRoutes(r, db, log)
	pullrequestRouter.RegisterRoutes(r, db, log)

	serverPort := getEnv("SERVER_PORT", ":8080")
	if err := r.Run(serverPort); err != nil {
		log.Fatalw("failed to start server", "error", err)
	}
}
