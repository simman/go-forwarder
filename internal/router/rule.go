package router

import "net/http"

// Rule represents a matching rule interface
type Rule interface {
	Match(req *http.Request) bool
}

// AndRule combines two rules with AND logic
type AndRule struct {
	Left  Rule
	Right Rule
}

// Match returns true if both left and right rules match
func (r *AndRule) Match(req *http.Request) bool {
	return r.Left.Match(req) && r.Right.Match(req)
}

// OrRule combines two rules with OR logic
type OrRule struct {
	Left  Rule
	Right Rule
}

// Match returns true if either left or right rule matches
func (r *OrRule) Match(req *http.Request) bool {
	return r.Left.Match(req) || r.Right.Match(req)
}

// NotRule negates a rule
type NotRule struct {
	Inner Rule
}

// Match returns the opposite of the inner rule's match
func (r *NotRule) Match(req *http.Request) bool {
	return !r.Inner.Match(req)
}
