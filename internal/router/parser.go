package router

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/simman/go-forwarder/internal/router/matchers"
)

// ParseRule parses a rule string into a Rule object
func ParseRule(ruleStr string) (Rule, error) {
	p := &parser{
		input: strings.TrimSpace(ruleStr),
		pos:   0,
	}
	return p.parse()
}

type parser struct {
	input string
	pos   int
}

// parse is the entry point for parsing
func (p *parser) parse() (Rule, error) {
	return p.parseOr()
}

// parseOr handles OR operations (lowest precedence)
func (p *parser) parseOr() (Rule, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if !p.matchString("||") {
			break
		}
		p.pos += 2
		p.skipWhitespace()

		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrRule{Left: left, Right: right}
	}

	return left, nil
}

// parseAnd handles AND operations
func (p *parser) parseAnd() (Rule, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if !p.matchString("&&") {
			break
		}
		p.pos += 2
		p.skipWhitespace()

		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &AndRule{Left: left, Right: right}
	}

	return left, nil
}

// parseUnary handles NOT operations and parentheses
func (p *parser) parseUnary() (Rule, error) {
	p.skipWhitespace()

	// Handle NOT
	if p.matchChar('!') {
		p.pos++
		p.skipWhitespace()
		inner, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &NotRule{Inner: inner}, nil
	}

	// Handle parentheses
	if p.matchChar('(') {
		p.pos++
		p.skipWhitespace()
		rule, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if !p.matchChar(')') {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		p.pos++
		return rule, nil
	}

	// Parse primary matcher
	return p.parseMatcher()
}

// parseMatcher parses individual matchers like Host{example.com}
func (p *parser) parseMatcher() (Rule, error) {
	p.skipWhitespace()

	// Find matcher name
	nameStart := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '{' && p.input[p.pos] != ' ' {
		p.pos++
	}

	if nameStart == p.pos {
		return nil, fmt.Errorf("expected matcher name at position %d", p.pos)
	}

	name := p.input[nameStart:p.pos]
	p.skipWhitespace()

	// Expect opening brace
	if !p.matchChar('{') {
		return nil, fmt.Errorf("expected '{' after matcher name at position %d", p.pos)
	}
	p.pos++

	// Find closing brace
	valueStart := p.pos
	depth := 1
	for p.pos < len(p.input) && depth > 0 {
		if p.input[p.pos] == '{' {
			depth++
		} else if p.input[p.pos] == '}' {
			depth--
		}
		if depth > 0 {
			p.pos++
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("unmatched braces at position %d", p.pos)
	}

	value := p.input[valueStart:p.pos]
	p.pos++ // Skip closing brace

	// Create matcher based on name
	return p.createMatcher(name, value)
}

// createMatcher creates a matcher based on the name and value
func (p *parser) createMatcher(name, value string) (Rule, error) {
	switch name {
	case "Host":
		return &matchers.HostMatcher{Pattern: value}, nil

	case "Path":
		return &matchers.PathMatcher{Path: value}, nil

	case "PathPrefix":
		return &matchers.PathPrefixMatcher{Prefix: value}, nil

	case "Method":
		methods := strings.Split(value, ",")
		for i := range methods {
			methods[i] = strings.TrimSpace(methods[i])
		}
		return &matchers.MethodMatcher{Methods: methods}, nil

	case "Header":
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Header matcher format, expected Key=Value")
		}
		return &matchers.HeaderMatcher{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		}, nil

	case "HeaderRegex":
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid HeaderRegex matcher format, expected Key=Pattern")
		}
		pattern, err := regexp.Compile(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
		return &matchers.HeaderRegexMatcher{
			Key:     strings.TrimSpace(parts[0]),
			Pattern: pattern,
		}, nil

	case "Query":
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid Query matcher format, expected Key=Value")
		}
		return &matchers.QueryMatcher{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		}, nil

	default:
		return nil, fmt.Errorf("unknown matcher: %s", name)
	}
}

// skipWhitespace skips whitespace characters
func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n' || p.input[p.pos] == '\r') {
		p.pos++
	}
}

// matchChar checks if the current character matches
func (p *parser) matchChar(ch byte) bool {
	return p.pos < len(p.input) && p.input[p.pos] == ch
}

// matchString checks if the current position matches the string
func (p *parser) matchString(s string) bool {
	return p.pos+len(s) <= len(p.input) && p.input[p.pos:p.pos+len(s)] == s
}
