package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kbaljak/triAGe/internal/session"
)

// sessionFilter is a parsed advanced-filter query: an optional age
// constraint and/or an optional regex, both applied to Session.Title +
// Session.Project.
type sessionFilter struct {
	pattern *regexp.Regexp
	days    int
	older   bool // true: "days>N" (older than N days); false: "days<N" (newer than)
	hasAge  bool
}

// daysToken matches a "days>N" or "days<N" token anywhere in the query,
// case-insensitively and with optional spaces around the operator.
var daysToken = regexp.MustCompile(`(?i)days\s*([<>])\s*(\d+)`)

// parseSessionFilter parses a query like "days>7 kubernetes" into a
// sessionFilter. The days>N / days<N token, if present, is stripped out;
// whatever text remains becomes a case-insensitive regex. An empty query
// (or one that's only whitespace after stripping the token) yields a
// sessionFilter that matches everything.
func parseSessionFilter(query string) (sessionFilter, error) {
	var f sessionFilter

	rest := query
	if loc := daysToken.FindStringSubmatchIndex(rest); loc != nil {
		n, err := strconv.Atoi(rest[loc[4]:loc[5]])
		if err != nil {
			return f, fmt.Errorf("invalid days value: %w", err)
		}
		f.hasAge = true
		f.days = n
		f.older = rest[loc[2]:loc[3]] == ">"
		rest = rest[:loc[0]] + rest[loc[1]:]
	}

	rest = strings.Join(strings.Fields(rest), " ") // collapse whitespace left by removing the token
	if rest != "" {
		re, err := regexp.Compile("(?i)" + rest)
		if err != nil {
			return f, fmt.Errorf("invalid pattern: %w", err)
		}
		f.pattern = re
	}
	return f, nil
}

// matches reports whether a session satisfies the filter.
func (f sessionFilter) matches(s session.Session) bool {
	if f.hasAge {
		age := time.Since(s.UpdatedAt)
		cutoff := time.Duration(f.days) * 24 * time.Hour
		if f.older && age < cutoff {
			return false
		}
		if !f.older && age >= cutoff {
			return false
		}
	}
	if f.pattern != nil && !f.pattern.MatchString(s.Title+" "+s.Project) {
		return false
	}
	return true
}

func (f sessionFilter) isEmpty() bool {
	return !f.hasAge && f.pattern == nil
}
