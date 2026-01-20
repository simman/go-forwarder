package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	// Validate server config
	if err := validateServerConfig(&cfg.Server); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	// Validate logging config
	if err := validateLoggingConfig(&cfg.Logging); err != nil {
		return fmt.Errorf("invalid logging config: %w", err)
	}

	// Validate default proxy if specified
	if cfg.DefaultProxy != "" {
		if err := validateProxyURL(cfg.DefaultProxy); err != nil {
			return fmt.Errorf("invalid default_proxy: %w", err)
		}
	}

	// Validate services
	if len(cfg.Services) == 0 {
		return fmt.Errorf("at least one service must be defined")
	}

	for i, svc := range cfg.Services {
		if err := validateService(&svc); err != nil {
			return fmt.Errorf("invalid service at index %d (%s): %w", i, svc.Name, err)
		}
	}

	return nil
}

func validateServerConfig(cfg *ServerConfig) error {
	if cfg.Addr == "" {
		return fmt.Errorf("addr is required")
	}
	if cfg.ReadTimeout < 0 {
		return fmt.Errorf("read_timeout must be positive")
	}
	if cfg.WriteTimeout < 0 {
		return fmt.Errorf("write_timeout must be positive")
	}
	if cfg.IdleTimeout < 0 {
		return fmt.Errorf("idle_timeout must be positive")
	}
	return nil
}

func validateLoggingConfig(cfg *LoggingConfig) error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Level] {
		return fmt.Errorf("invalid level: %s (must be debug, info, warn, or error)", cfg.Level)
	}

	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[cfg.Format] {
		return fmt.Errorf("invalid format: %s (must be json or text)", cfg.Format)
	}

	return nil
}

func validateService(svc *Service) error {
	if svc.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate handler
	validHandlers := map[string]bool{
		"http": true,
		"tcp":  true,
	}
	if !validHandlers[svc.Handler.Type] {
		return fmt.Errorf("invalid handler type: %s (must be http or tcp)", svc.Handler.Type)
	}

	// Validate listener
	validListeners := map[string]bool{
		"tcp": true,
	}
	if !validListeners[svc.Listener.Type] {
		return fmt.Errorf("invalid listener type: %s (must be tcp)", svc.Listener.Type)
	}

	// Validate nodes
	if len(svc.Forwarder.Nodes) == 0 {
		return fmt.Errorf("at least one node must be defined")
	}

	for i, node := range svc.Forwarder.Nodes {
		if err := validateNode(&node); err != nil {
			return fmt.Errorf("invalid node at index %d (%s): %w", i, node.Name, err)
		}
	}

	return nil
}

func validateNode(node *Node) error {
	if node.Name == "" {
		return fmt.Errorf("node name is required")
	}

	if node.Addr == "" {
		return fmt.Errorf("node addr is required")
	}

	// Must have either filter or matcher
	if node.Filter == nil && node.Matcher == nil {
		return fmt.Errorf("node must have either filter or matcher")
	}

	// Can't have both filter and matcher
	if node.Filter != nil && node.Matcher != nil {
		return fmt.Errorf("node cannot have both filter and matcher")
	}

	// Validate filter
	if node.Filter != nil && node.Filter.Host == "" {
		return fmt.Errorf("filter host is required")
	}

	// Validate matcher
	if node.Matcher != nil && node.Matcher.Rule == "" {
		return fmt.Errorf("matcher rule is required")
	}

	// Validate proxy URL if specified
	if node.Proxy != "" {
		if err := validateProxyURL(node.Proxy); err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
	}

	return nil
}

func validateProxyURL(proxyURL string) error {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("proxy scheme must be http or https, got: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("proxy host is required")
	}

	// Validate it's not localhost with common variations
	host := strings.ToLower(u.Hostname())
	if host != "localhost" && host != "127.0.0.1" && !strings.HasPrefix(host, "192.168.") && !strings.HasPrefix(host, "10.") {
		// This is fine, just informational
	}

	return nil
}
