package router

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/simman/go-forwarder/internal/config"
	"github.com/simman/go-forwarder/internal/router/matchers"
)

// Router routes requests to backend nodes based on matching rules
type Router struct {
	routes []Route
	mu     sync.RWMutex
}

// Route represents a routing rule with its associated node
type Route struct {
	Name string
	Rule Rule
	Node *config.Node
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make([]Route, 0),
	}
}

// UpdateRoutes updates the routing table from configuration
func (r *Router) UpdateRoutes(services []config.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var routes []Route

	for _, svc := range services {
		for _, node := range svc.Forwarder.Nodes {
			route, err := r.buildRoute(&node)
			if err != nil {
				return fmt.Errorf("failed to build route for node %s: %w", node.Name, err)
			}
			routes = append(routes, route)
		}
	}

	r.routes = routes
	log.Info().Int("count", len(routes)).Msg("routes updated")

	return nil
}

// buildRoute creates a Route from a Node configuration
func (r *Router) buildRoute(node *config.Node) (Route, error) {
	var rule Rule
	var err error

	// Use filter (simple host matching) if specified
	if node.Filter != nil {
		rule = &matchers.HostMatcher{Pattern: node.Filter.Host}
	} else if node.Matcher != nil {
		// Use matcher (complex rule) if specified
		rule, err = ParseRule(node.Matcher.Rule)
		if err != nil {
			return Route{}, fmt.Errorf("failed to parse rule: %w", err)
		}
	} else {
		return Route{}, fmt.Errorf("node must have either filter or matcher")
	}

	return Route{
		Name: node.Name,
		Rule: rule,
		Node: node,
	}, nil
}

// Match finds the first matching route for the request
func (r *Router) Match(req *http.Request) (*config.Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.Rule.Match(req) {
			log.Debug().
				Str("route", route.Name).
				Str("host", req.Host).
				Str("path", req.URL.Path).
				Msg("route matched")
			return route.Node, true
		}
	}

	log.Debug().
		Str("host", req.Host).
		Str("path", req.URL.Path).
		Msg("no route matched")

	return nil, false
}

// GetRoutes returns all configured routes (for debugging/monitoring)
func (r *Router) GetRoutes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]Route, len(r.routes))
	copy(routes, r.routes)
	return routes
}
