package matchers

import (
	"net/http"
	"strings"
)

// HostMatcher matches requests based on the Host header
type HostMatcher struct {
	Pattern string
}

// Match checks if the request matches the host pattern
func (m *HostMatcher) Match(req *http.Request) bool {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Exact match
	if m.Pattern == host {
		return true
	}

	// Wildcard match (*.example.com)
	if strings.HasPrefix(m.Pattern, "*.") {
		domain := m.Pattern[2:] // Remove "*."
		return strings.HasSuffix(host, "."+domain) || host == domain
	}

	return false
}
