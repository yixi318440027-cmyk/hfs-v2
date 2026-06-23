package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// VFSConfig holds the VFS configuration.
type VFSConfig struct {
	Roots []VFSRoot `yaml:"roots" json:"roots"`
}

// VFSRoot defines a single virtual file system root.
type VFSRoot struct {
	Path     string `yaml:"path" json:"path"`           // local filesystem path
	Name     string `yaml:"name" json:"name"`           // display name
	Public   bool   `yaml:"public" json:"public"`       // accessible without login
	ReadOnly bool   `yaml:"read_only" json:"read_only"` // read-only access
}

// Config holds the application configuration.
type Config struct {
	Port      string    `yaml:"port" json:"port"`           // HTTP server listen port, e.g. ":8080"
	DataDir   string    `yaml:"data_dir" json:"data_dir"`   // Data directory for database and persistent files
	LogLevel  string    `yaml:"log_level" json:"log_level"` // Logging level: debug, info, warn, error
	VFS       VFSConfig `yaml:"vfs" json:"vfs"`             // VFS configuration
	JWTSecret string    `yaml:"jwt_secret" json:"jwt_secret"` // JWT signing secret
	AdminUser string    `yaml:"admin_user" json:"admin_user"` // Default admin username
	AdminPass string    `yaml:"admin_pass" json:"admin_pass"` // Default admin password
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Port:      ":8080",
		DataDir:   "./data",
		LogLevel:  "info",
		JWTSecret: "change-me-in-production",
		AdminUser: "admin",
		AdminPass: "admin",
		VFS: VFSConfig{
			Roots: []VFSRoot{
				{Path: "./files", Name: "Files", Public: true, ReadOnly: false},
			},
		},
	}
}

// LoadConfig reads and parses a YAML configuration file at the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
