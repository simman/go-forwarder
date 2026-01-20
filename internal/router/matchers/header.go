package matchers

import (
	"net/http"
	"regexp"
)

// HeaderMatcher matches requests based on header key-value pairs
type HeaderMatcher struct {
	Key   string
	Value string
}

// Match checks if the request has the specified header with the exact value
func (m *HeaderMatcher) Match(req *http.Request) bool {
	headerValue := req.Header.Get(m.Key)
	return headerValue == m.Value
}

// HeaderRegexMatcher matches requests based on header key and value regex pattern
type HeaderRegexMatcher struct {
	Key     string
	Pattern *regexp.Regexp
}

// Match checks if the request header matches the regex pattern
func (m *HeaderRegexMatcher) Match(req *http.Request) bool {
	headerValue := req.Header.Get(m.Key)
	if headerValue == "" {
		return false
	}
	return m.Pattern.MatchString(headerValue)
}
