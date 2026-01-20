package config

import "time"

// Config represents the entire application configuration
type Config struct {
	Server       ServerConfig   `yaml:"server"`
	Logging      LoggingConfig  `yaml:"logging"`
	DefaultProxy string         `yaml:"default_proxy"`
	Services     []Service      `yaml:"services"`
}

// ServerConfig contains global server settings
type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
	Output string `yaml:"output"` // stdout, stderr, or file path
}

// Service represents a service configuration
type Service struct {
	Name      string    `yaml:"name"`
	Addr      string    `yaml:"addr,omitempty"`
	Handler   Handler   `yaml:"handler"`
	Listener  Listener  `yaml:"listener"`
	Forwarder Forwarder `yaml:"forwarder"`
}

// Handler defines the handler type and metadata
type Handler struct {
	Type     string         `yaml:"type"`
	Metadata map[string]any `yaml:"metadata,omitempty"`
}

// Listener defines the listener type
type Listener struct {
	Type string `yaml:"type"`
}

// Forwarder contains forwarding configuration
type Forwarder struct {
	Nodes []Node `yaml:"nodes"`
}

// Node represents a forwarding node with routing rules
type Node struct {
	Name    string   `yaml:"name"`
	Addr    string   `yaml:"addr"`
	Filter  *Filter  `yaml:"filter,omitempty"`
	Matcher *Matcher `yaml:"matcher,omitempty"`
	Proxy   string   `yaml:"proxy,omitempty"`
}

// Filter provides simple host-based filtering
type Filter struct {
	Host string `yaml:"host"`
}

// Matcher provides advanced rule-based matching
type Matcher struct {
	Rule string `yaml:"rule"`
}
