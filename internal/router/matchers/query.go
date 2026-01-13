package matchers

import "net/http"

// QueryMatcher matches requests based on query parameter key-value pairs
type QueryMatcher struct {
	Key   string
	Value string
}

// Match checks if the request has the specified query parameter with the exact value
func (m *QueryMatcher) Match(req *http.Request) bool {
	queryValue := req.URL.Query().Get(m.Key)
	return queryValue == m.Value
}
