package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Admin    AdminConfig    `yaml:"admin"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type JWTConfig struct {
	Secret string        `yaml:"secret"`
	Expiry time.Duration `yaml:"expiry"`
}

type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`
}

var AppConfig *Config

func Load(path string) (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: 8989,
			Host: "0.0.0.0",
		},
		Database: DatabaseConfig{
			Path: "./data/vps-panel.db",
		},
		JWT: JWTConfig{
			Secret: "change-this-secret-in-production",
			Expiry: 24 * time.Hour,
		},
		Admin: AdminConfig{
			Username: "admin",
			Password: "admin123",
			Email:    "admin@localhost",
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If config file doesn't exist, try to look in the same directory as the executable
			if ex, err := os.Executable(); err == nil {
				exeDir := filepath.Dir(ex)
				config.Database.Path = filepath.Join(exeDir, "data", "vps-panel.db")
			}
			AppConfig = config
			return config, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Resolve relative paths
	if !filepath.IsAbs(config.Database.Path) {
		if ex, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(ex)
			config.Database.Path = filepath.Join(exeDir, config.Database.Path)
		}
	}

	// Override with environment variables
	if port := os.Getenv("VPS_PANEL_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			config.Server.Port = p
		}
	}
	if secret := os.Getenv("VPS_PANEL_JWT_SECRET"); secret != "" {
		config.JWT.Secret = secret
	}

	AppConfig = config
	return config, nil
}
