// Package session defines the provider-agnostic model for AI agent sessions:
// discovering where each agent's CLI stores its conversation history on
// disk, listing it, resuming it, and deleting entries.
package session

import (
	"fmt"
	"time"
)

// Session represents a single conversation/session belonging to one agent.
type Session struct {
	ID        string    // stable identifier (usually a filename or UUID)
	Title     string    // best-effort human-readable summary
	Project   string    // working directory / workspace the session ran in, if known
	UpdatedAt time.Time // last activity time, used for sorting and display
	SizeBytes int64     // best-effort on-disk size across all files for this session

	// Paths are every file/directory on disk that make up this session.
	// Deleting a session removes all of them. Always non-empty.
	Paths []string
}

// Provider knows how to discover and manage sessions for one AI agent/CLI.
type Provider interface {
	// ID is a short stable key, e.g. "claude".
	ID() string
	// Name is the human-readable agent name, e.g. "Claude Code".
	Name() string
	// Detect reports whether this agent's data directory exists on this
	// machine at all (regardless of whether it currently has sessions).
	Detect() bool
	// ListSessions returns every session found for this agent, unsorted.
	ListSessions() ([]Session, error)
	// DeleteSession permanently removes a session and its files on disk.
	DeleteSession(s Session) error
}

// ErrResumeUnsupported means this agent has no known way to resume a
// session from the command line. Returned by Resumable.ResumeCommand.
var ErrResumeUnsupported = fmt.Errorf("resuming isn't supported for this agent yet")

// Resumable is implemented by providers that know how to hand control back
// to the underlying agent CLI so the user can continue a session
// interactively. Not every provider implements it — Antigravity and Copilot,
// for instance, aren't driven through a resumable terminal command the way
// Claude Code and Codex are.
type Resumable interface {
	// ResumeCommand returns the argv (argv[0] is the executable, looked up
	// on PATH) and working directory to run in order to resume s
	// interactively, attached to the current terminal. err is
	// ErrResumeUnsupported (or wraps it) if this agent can't do that.
	ResumeCommand(s Session) (argv []string, dir string, err error)
}
