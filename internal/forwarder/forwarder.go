package forwarder

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/simman/go-forwarder/internal/config"
	"golang.org/x/net/http2"
)

// Forwarder forwards requests to backend servers through a proxy
type Forwarder struct {
	clients map[string]*http.Client // keyed by proxy URL
}

// NewForwarder creates a new forwarder
func NewForwarder() *Forwarder {
	return &Forwarder{
		clients: make(map[string]*http.Client),
	}
}

// Forward forwards the request to the target node
func (f *Forwarder) Forward(w http.ResponseWriter, r *http.Request, node *config.Node) error {
	// Get or create HTTP client for this proxy
	client, err := f.getClient(node.Proxy)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// Build target URL
	targetURL := f.buildTargetURL(r, node)

	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Copy headers
	copyHeaders(proxyReq.Header, r.Header)

	// Set proper host header
	proxyReq.Host = node.Addr
	if idx := len(node.Addr) - 1; idx >= 0 && node.Addr[idx] >= '0' && node.Addr[idx] <= '9' {
		// If addr ends with port number, strip it for host header
		if colonIdx := len(node.Addr) - 1; colonIdx >= 0 {
			for colonIdx >= 0 && node.Addr[colonIdx] != ':' {
				colonIdx--
			}
			if colonIdx > 0 {
				proxyReq.Host = node.Addr[:colonIdx]
			}
		}
	}

	// Perform request
	start := time.Now()
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("target", targetURL).
			Str("node", node.Name).
			Msg("request failed")
		return fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Log request
	log.Info().
		Str("method", r.Method).
		Str("host", r.Host).
		Str("path", r.URL.Path).
		Str("node", node.Name).
		Str("target", targetURL).
		Int("status", resp.StatusCode).
		Dur("duration", duration).
		Msg("request forwarded")

	// Copy response headers
	copyHeaders(w.Header(), resp.Header)

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to copy response body")
		return fmt.Errorf("failed to copy response: %w", err)
	}

	return nil
}

// buildTargetURL constructs the target URL from request and node
func (f *Forwarder) buildTargetURL(r *http.Request, node *config.Node) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	// Use node.Addr which includes host:port
	return fmt.Sprintf("%s://%s%s", scheme, node.Addr, r.URL.RequestURI())
}

// getClient returns or creates an HTTP client for the given proxy URL
func (f *Forwarder) getClient(proxyURL string) (*http.Client, error) {
	if proxyURL == "" {
		proxyURL = "direct" // special key for direct connection
	}

	if client, ok := f.clients[proxyURL]; ok {
		return client, nil
	}

	// Create new client
	client, err := createClient(proxyURL)
	if err != nil {
		return nil, err
	}

	f.clients[proxyURL] = client
	return client, nil
}

// createClient creates a new HTTP client with the specified proxy
func createClient(proxyURL string) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	// Configure proxy if specified
	if proxyURL != "" && proxyURL != "direct" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxy)
	}

	// Enable HTTP/2
	if err := http2.ConfigureTransport(transport); err != nil {
		log.Warn().Err(err).Msg("failed to configure HTTP/2 transport")
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects
			return http.ErrUseLastResponse
		},
	}, nil
}

// copyHeaders copies HTTP headers from src to dst
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Close closes all HTTP clients
func (f *Forwarder) Close() error {
	for _, client := range f.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}
