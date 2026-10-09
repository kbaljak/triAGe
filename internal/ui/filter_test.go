package ui

import (
	"testing"
	"time"

	"github.com/kbaljak/triAGe/internal/session"
)

func sessionAt(title string, age time.Duration) session.Session {
	return session.Session{Title: title, Project: "/some/project", UpdatedAt: time.Now().Add(-age)}
}

func TestParseSessionFilter(t *testing.T) {
	cases := []struct {
		name    string
		query   string
		session session.Session
		want    bool
	}{
		{"empty query matches everything", "", sessionAt("anything", 0), true},
		{"plain regex matches", "kube", sessionAt("Kubernetes cluster fix", 0), true},
		{"plain regex is case-insensitive", "KUBE", sessionAt("kubernetes cluster fix", 0), true},
		{"plain regex non-match", "kube", sessionAt("unrelated session", 0), false},
		{"days> excludes recent session", "days>7", sessionAt("x", 2*24*time.Hour), false},
		{"days> includes older session", "days>7", sessionAt("x", 10*24*time.Hour), true},
		{"days< includes recent session", "days<7", sessionAt("x", 2*24*time.Hour), true},
		{"days< excludes older session", "days<7", sessionAt("x", 10*24*time.Hour), false},
		{"combined age and pattern both must match", "days>7 kube", sessionAt("kubernetes fix", 10*24*time.Hour), true},
		{"combined: age matches but pattern doesn't", "days>7 kube", sessionAt("unrelated", 10*24*time.Hour), false},
		{"combined: pattern matches but age doesn't", "days>7 kube", sessionAt("kubernetes fix", 1*24*time.Hour), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, err := parseSessionFilter(c.query)
			if err != nil {
				t.Fatalf("parseSessionFilter(%q) returned error: %v", c.query, err)
			}
			if got := f.matches(c.session); got != c.want {
				t.Errorf("parseSessionFilter(%q).matches(%+v) = %v, want %v", c.query, c.session, got, c.want)
			}
		})
	}
}

// TestSessionFilterIgnoresProjectPath locks in a real bug report: matching
// against the full project path meant a bare letter anywhere in it (e.g.
// from the username in a home directory) made almost any pattern match
// almost every session, regardless of title.
func TestSessionFilterIgnoresProjectPath(t *testing.T) {
	s := session.Session{Title: "Kubernetes CI/CD utility", Project: "/home/kbaljak/Documents/project"}

	f, err := parseSessionFilter("ju*") // "j" + zero-or-more "u" — matches the "j" in "kbaljak"
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.matches(s) {
		t.Error("pattern matched via the project path, not the title — project should not be searched")
	}

	jupyter := session.Session{Title: "Jupyter notebook debug", Project: "/home/kbaljak/Documents/project"}
	if !f.matches(jupyter) {
		t.Error("expected the pattern to still match a title that actually contains it")
	}
}

func TestParseSessionFilterInvalidRegex(t *testing.T) {
	if _, err := parseSessionFilter("(unclosed"); err == nil {
		t.Error("expected an error for an invalid regex, got nil")
	}
}

func TestSessionFilterIsEmpty(t *testing.T) {
	f, err := parseSessionFilter("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.isEmpty() {
		t.Error("expected an empty query to produce an empty filter")
	}

	f, err = parseSessionFilter("days>1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.isEmpty() {
		t.Error("expected a days-only query to not be empty")
	}
}
