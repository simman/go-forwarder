package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/simman/go-forwarder/internal/config"
	"github.com/simman/go-forwarder/internal/forwarder"
	"github.com/simman/go-forwarder/internal/router"
)

// Server represents the main proxy server
type Server struct {
	config    *config.Config
	router    *router.Router
	forwarder *forwarder.Forwarder
	servers   []*http.Server
	mu        sync.RWMutex
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	s := &Server{
		config:    cfg,
		router:    router.NewRouter(),
		forwarder: forwarder.NewForwarder(),
		servers:   make([]*http.Server, 0),
	}

	// Initialize routes
	if err := s.router.UpdateRoutes(cfg.Services); err != nil {
		return nil, fmt.Errorf("failed to initialize routes: %w", err)
	}

	return s, nil
}

// Start starts all configured servers
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create HTTP servers for each unique address
	addrs := s.getUniqueAddresses()

	for _, addr := range addrs {
		srv := &http.Server{
			Addr:         addr,
			Handler:      s,
			ReadTimeout:  s.config.Server.ReadTimeout,
			WriteTimeout: s.config.Server.WriteTimeout,
			IdleTimeout:  s.config.Server.IdleTimeout,
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}

		s.servers = append(s.servers, srv)

		go func(srv *http.Server, addr string) {
			log.Info().Str("addr", addr).Msg("server started")
			if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Error().Err(err).Str("addr", addr).Msg("server error")
			}
		}(srv, addr)
	}

	return nil
}

// Stop gracefully stops all servers
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Info().Msg("stopping servers")

	var wg sync.WaitGroup
	errCh := make(chan error, len(s.servers))

	for _, srv := range s.servers {
		wg.Add(1)
		go func(srv *http.Server) {
			defer wg.Done()
			if err := srv.Shutdown(ctx); err != nil {
				errCh <- err
			}
		}(srv)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	// Close forwarder
	if err := s.forwarder.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	log.Info().Msg("servers stopped")
	return nil
}

// ServeHTTP handles incoming HTTP requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle CONNECT method for HTTPS proxying
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
		return
	}

	// Check for WebSocket upgrade
	if isWebSocketUpgrade(r) {
		s.handleWebSocket(w, r)
		return
	}

	// Handle regular HTTP request
	s.handleHTTP(w, r)
}

// Reload reloads the configuration
func (s *Server) Reload(cfg *config.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update router with new configuration
	if err := s.router.UpdateRoutes(cfg.Services); err != nil {
		return fmt.Errorf("failed to update routes: %w", err)
	}

	s.config = cfg

	log.Info().Msg("configuration reloaded")
	return nil
}

// getUniqueAddresses returns unique server addresses from config
func (s *Server) getUniqueAddresses() []string {
	addrs := make(map[string]bool)

	// Add global server address
	addrs[s.config.Server.Addr] = true

	// Add service-specific addresses
	for _, svc := range s.config.Services {
		if svc.Addr != "" {
			addrs[svc.Addr] = true
		}
	}

	result := make([]string, 0, len(addrs))
	for addr := range addrs {
		result = append(result, addr)
	}

	return result
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func isWebSocketUpgrade(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket" &&
		r.Header.Get("Connection") == "Upgrade"
}
