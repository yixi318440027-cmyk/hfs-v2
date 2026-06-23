package config

// Config holds the application configuration.
type Config struct {
	Port     string // HTTP server listen port, e.g. ":8080"
	DataDir  string // Data directory for database and persistent files
	LogLevel string // Logging level: debug, info, warn, error
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Port:     ":8080",
		DataDir:  "./data",
		LogLevel: "info",
	}
}
