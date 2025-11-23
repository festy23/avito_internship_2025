package config

import "fmt"

// Config holds application configuration.
type Config struct {
	// Server holds HTTP server configuration.
	Server ServerConfig
	// Logger holds logger configuration.
	Logger LoggerConfig
	// GinMode is the Gin framework mode (debug, release, test).
	GinMode string
}

// LoadFromEnv loads all configuration from environment variables.
func LoadFromEnv() Config {
	return Config{
		Server:  LoadServerConfigFromEnv(),
		Logger:  LoadLoggerConfigFromEnv(),
		GinMode: GetEnv("GIN_MODE", "release"),
	}
}

// Validate validates all configuration.
func (c Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config validation failed: %w", err)
	}

	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("logger config validation failed: %w", err)
	}

	validGinModes := map[string]bool{
		"debug":   true,
		"release": true,
		"test":    true,
	}
	if !validGinModes[c.GinMode] {
		return fmt.Errorf("invalid GIN_MODE: %s (must be: debug, release, test)", c.GinMode)
	}

	return nil
}
