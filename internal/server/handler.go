package server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// handleHTTP handles regular HTTP requests
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Find matching route
	node, matched := s.router.Match(r)
	if !matched {
		s.handleNoMatch(w, r)
		return
	}

	// Forward request
	if err := s.forwarder.Forward(w, r, node); err != nil {
		log.Error().
			Err(err).
			Str("host", r.Host).
			Str("path", r.URL.Path).
			Str("node", node.Name).
			Msg("failed to forward request")
		s.handleError(w, r, http.StatusBadGateway, "failed to forward request")
		return
	}
}

// handleNoMatch handles requests that don't match any route
func (s *Server) handleNoMatch(w http.ResponseWriter, r *http.Request) {
	log.Warn().
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Msg("no matching route found")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadGateway)

	response := map[string]string{
		"error":  "no matching route found",
		"host":   r.Host,
		"path":   r.URL.Path,
		"method": r.Method,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("failed to encode error response")
	}
}

// handleError handles error responses
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error": message,
		"host":  r.Host,
		"path":  r.URL.Path,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("failed to encode error response")
	}
}
