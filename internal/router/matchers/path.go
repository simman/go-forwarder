package matchers

import (
	"net/http"
	"strings"
)

// PathMatcher matches requests based on exact path
type PathMatcher struct {
	Path string
}

// Match checks if the request path matches exactly
func (m *PathMatcher) Match(req *http.Request) bool {
	return req.URL.Path == m.Path
}

// PathPrefixMatcher matches requests based on path prefix
type PathPrefixMatcher struct {
	Prefix string
}

// Match checks if the request path starts with the prefix
func (m *PathPrefixMatcher) Match(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, m.Prefix)
}
