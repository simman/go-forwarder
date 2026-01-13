package server

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

// handleConnect handles HTTPS CONNECT requests for tunneling
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	// Match route based on host
	node, matched := s.router.Match(r)
	if !matched {
		log.Warn().
			Str("host", r.Host).
			Msg("no matching route for CONNECT")
		http.Error(w, "No matching route found", http.StatusBadGateway)
		return
	}

	log.Debug().
		Str("host", r.Host).
		Str("node", node.Name).
		Msg("handling CONNECT request")

	// Connect to proxy or directly to target
	var targetConn net.Conn
	var err error

	if node.Proxy != "" {
		// Connect through proxy
		targetConn, err = s.connectThroughProxy(node.Proxy, node.Addr)
	} else {
		// Connect directly
		targetConn, err = net.DialTimeout("tcp", node.Addr, 30*time.Second)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("host", r.Host).
			Str("node", node.Name).
			Msg("failed to connect to target")
		http.Error(w, "Failed to connect to target", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Error().Msg("ResponseWriter does not support hijacking")
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Error().Err(err).Msg("failed to hijack connection")
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established to client
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Error().Err(err).Msg("failed to send connection established")
		return
	}

	// Start bidirectional copy
	log.Info().
		Str("host", r.Host).
		Str("node", node.Name).
		Msg("CONNECT tunnel established")

	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(targetConn, clientConn)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(clientConn, targetConn)
		errCh <- err
	}()

	// Wait for one direction to finish
	err = <-errCh
	if err != nil && err != io.EOF {
		log.Debug().Err(err).Msg("tunnel copy error")
	}

	log.Debug().
		Str("host", r.Host).
		Str("node", node.Name).
		Msg("CONNECT tunnel closed")
}

// connectThroughProxy connects to the target through an HTTP proxy
func (s *Server) connectThroughProxy(proxyURL, targetAddr string) (net.Conn, error) {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	// Connect to proxy
	proxyConn, err := net.DialTimeout("tcp", proxy.Host, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// Send CONNECT request to proxy
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetAddr, targetAddr)
	_, err = proxyConn.Write([]byte(connectReq))
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to send CONNECT to proxy: %w", err)
	}

	// Read response from proxy
	buf := make([]byte, 1024)
	n, err := proxyConn.Read(buf)
	if err != nil {
		proxyConn.Close()
		return nil, fmt.Errorf("failed to read proxy response: %w", err)
	}

	// Check for 200 Connection Established
	response := string(buf[:n])
	if len(response) < 12 || response[9:12] != "200" {
		proxyConn.Close()
		return nil, fmt.Errorf("proxy returned non-200 response: %s", response)
	}

	return proxyConn, nil
}
