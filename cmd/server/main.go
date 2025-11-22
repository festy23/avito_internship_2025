// Package main provides the entry point for the HTTP server.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/team"
	"github.com/festy23/avito_internship/internal/user"
)

// getEnv reads an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Build DSN from environment variables
	host := getEnv("DB_HOST", "localhost")
	dbUser := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "avito_internship")
	port := getEnv("DB_PORT", "5432")
	sslmode := getEnv("DB_SSLMODE", "disable")
	timezone := getEnv("DB_TIMEZONE", "UTC")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		host, dbUser, password, dbname, port, sslmode, timezone)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	r := gin.Default()

	team.RegisterRoutes(r, db)
	user.RegisterRoutes(r, db)

	serverPort := getEnv("SERVER_PORT", ":8080")
	if err := r.Run(serverPort); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
