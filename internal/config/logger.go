package config

import "fmt"

// LoggerConfig holds logger configuration.
type LoggerConfig struct {
	// Level is the logging level (debug, info, warn, error).
	Level string
	// Format is the logging format (json, console).
	Format string
	// Output is the output destination (stdout, stderr, or file path).
	Output string
}

// LoadLoggerConfigFromEnv loads logger configuration from environment variables.
func LoadLoggerConfigFromEnv() LoggerConfig {
	return LoggerConfig{
		Level:  GetEnv("LOG_LEVEL", "info"),
		Format: GetEnv("LOG_FORMAT", "json"),
		Output: GetEnv("LOG_OUTPUT", "stdout"),
	}
}

// Validate validates logger configuration.
func (c LoggerConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s (must be: debug, info, warn, error)", c.Level)
	}

	validFormats := map[string]bool{
		"json":    true,
		"console": true,
	}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s (must be: json, console)", c.Format)
	}

	return nil
}

// IsProduction returns true if logger is configured for production.
func (c LoggerConfig) IsProduction() bool {
	return c.Format == "json" && c.Level != "debug"
}
