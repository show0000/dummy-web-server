package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port int `yaml:"port"`
}

type JWTConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Secret            string `yaml:"secret"`
	AccessTokenExpiry string `yaml:"accessTokenExpiry"`
	RefreshTokenExpiry string `yaml:"refreshTokenExpiry"`
}

type PathsConfig struct {
	APIs    string `yaml:"apis"`
	Storage string `yaml:"storage"`
	Scripts string `yaml:"scripts"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	JWT    JWTConfig    `yaml:"jwt"`
	Paths  PathsConfig  `yaml:"paths"`
}

func (c *JWTConfig) AccessTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.AccessTokenExpiry)
}

func (c *JWTConfig) RefreshTokenDuration() (time.Duration, error) {
	return time.ParseDuration(c.RefreshTokenExpiry)
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{Port: 8080},
		JWT: JWTConfig{
			Enabled:           false,
			Secret:            "change-me-to-a-secure-secret",
			AccessTokenExpiry:  "15m",
			RefreshTokenExpiry: "168h",
		},
		Paths: PathsConfig{
			APIs:    "./apis.yaml",
			Storage: "./storage",
			Scripts: "./scripts",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}

	if c.JWT.Enabled {
		if c.JWT.Secret == "" {
			return fmt.Errorf("jwt.secret is required when jwt is enabled")
		}
		if _, err := c.JWT.AccessTokenDuration(); err != nil {
			return fmt.Errorf("jwt.accessTokenExpiry is invalid: %w", err)
		}
		if _, err := c.JWT.RefreshTokenDuration(); err != nil {
			return fmt.Errorf("jwt.refreshTokenExpiry is invalid: %w", err)
		}
	}

	if c.Paths.APIs == "" {
		return fmt.Errorf("paths.apis is required")
	}

	return nil
}
