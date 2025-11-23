package config

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	// Host is the server host (empty string means all interfaces).
	Host string
	// Port is the server port (e.g., ":8080" or "8080").
	Port string
	// ReadTimeout is the maximum duration for reading the entire request.
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out writes.
	WriteTimeout time.Duration
	// IdleTimeout is the maximum amount of time to wait for the next request.
	IdleTimeout time.Duration
}

// LoadServerConfigFromEnv loads server configuration from environment variables.
func LoadServerConfigFromEnv() ServerConfig {
	return ServerConfig{
		Host:         GetEnv("SERVER_HOST", ""),
		Port:         GetEnv("SERVER_PORT", ":8080"),
		ReadTimeout:  GetEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: GetEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  GetEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
	}
}

// GetAddress returns the full server address (host:port).
func (c ServerConfig) GetAddress() string {
	if c.Host == "" {
		return c.Port
	}

	// Remove leading colon from port if present, as net.JoinHostPort adds it
	port := strings.TrimPrefix(c.Port, ":")
	return net.JoinHostPort(c.Host, port)
}

// Validate validates server configuration.
func (c ServerConfig) Validate() error {
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("ReadTimeout must be greater than 0")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("WriteTimeout must be greater than 0")
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("IdleTimeout must be greater than 0")
	}
	return nil
}
