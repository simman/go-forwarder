package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if err := setDefaults(&cfg); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default values for optional fields
func setDefaults(cfg *Config) error {
	// Server defaults
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":22222"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 120 * time.Second
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	// Service defaults
	for i := range cfg.Services {
		svc := &cfg.Services[i]
		
		// Use global server addr if not specified for service
		if svc.Addr == "" {
			svc.Addr = cfg.Server.Addr
		}
		
		// Set default handler type
		if svc.Handler.Type == "" {
			svc.Handler.Type = "http"
		}
		
		// Set default listener type
		if svc.Listener.Type == "" {
			svc.Listener.Type = "tcp"
		}
		
		// Set node proxy defaults
		for j := range svc.Forwarder.Nodes {
			node := &svc.Forwarder.Nodes[j]
			if node.Proxy == "" && cfg.DefaultProxy != "" {
				node.Proxy = cfg.DefaultProxy
			}
		}
	}

	return nil
}
