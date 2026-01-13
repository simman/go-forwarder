package matchers

import (
	"net/http"
	"strings"
)

// MethodMatcher matches requests based on HTTP method
type MethodMatcher struct {
	Methods []string
}

// Match checks if the request method matches any of the allowed methods
func (m *MethodMatcher) Match(req *http.Request) bool {
	method := strings.ToUpper(req.Method)
	for _, allowed := range m.Methods {
		if strings.ToUpper(allowed) == method {
			return true
		}
	}
	return false
}
