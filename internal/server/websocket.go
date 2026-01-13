package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// handleWebSocket handles WebSocket upgrade requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Find matching route
	node, matched := s.router.Match(r)
	if !matched {
		log.Warn().
			Str("host", r.Host).
			Str("path", r.URL.Path).
			Msg("no matching route for WebSocket")
		http.Error(w, "No matching route found", http.StatusBadGateway)
		return
	}

	log.Debug().
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("node", node.Name).
		Msg("handling WebSocket upgrade")

	// Upgrade client connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to upgrade client connection")
		return
	}
	defer clientConn.Close()

	// Build backend WebSocket URL
	scheme := "wss"
	if r.TLS == nil {
		scheme = "ws"
	}
	backendURL := fmt.Sprintf("%s://%s%s", scheme, node.Addr, r.URL.RequestURI())

	// Create dialer with proxy support
	dialer := websocket.Dialer{
		HandshakeTimeout: upgrader.HandshakeTimeout,
	}

	if node.Proxy != "" {
		proxyURL, err := url.Parse(node.Proxy)
		if err != nil {
			log.Error().Err(err).Str("proxy", node.Proxy).Msg("invalid proxy URL")
			return
		}
		dialer.Proxy = http.ProxyURL(proxyURL)
	}

	// Connect to backend
	backendConn, resp, err := dialer.Dial(backendURL, r.Header)
	if err != nil {
		log.Error().
			Err(err).
			Str("url", backendURL).
			Msg("failed to connect to backend WebSocket")
		if resp != nil {
			log.Error().Int("status", resp.StatusCode).Msg("backend response status")
		}
		return
	}
	defer backendConn.Close()

	log.Info().
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("node", node.Name).
		Str("backend", backendURL).
		Msg("WebSocket connection established")

	// Bidirectional copy
	errCh := make(chan error, 2)

	// Client to backend
	go func() {
		errCh <- s.copyWebSocket(backendConn, clientConn, "client->backend")
	}()

	// Backend to client
	go func() {
		errCh <- s.copyWebSocket(clientConn, backendConn, "backend->client")
	}()

	// Wait for one direction to finish
	err = <-errCh
	if err != nil {
		log.Debug().Err(err).Msg("WebSocket copy error")
	}

	log.Debug().
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("node", node.Name).
		Msg("WebSocket connection closed")
}

// copyWebSocket copies messages from src to dst
func (s *Server) copyWebSocket(dst, src *websocket.Conn, direction string) error {
	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Debug().Err(err).Str("direction", direction).Msg("unexpected WebSocket close")
			}
			return err
		}

		err = dst.WriteMessage(messageType, message)
		if err != nil {
			log.Debug().Err(err).Str("direction", direction).Msg("failed to write WebSocket message")
			return err
		}
	}
}
