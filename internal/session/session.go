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
	// Home is the resolved config directory this instance reads from —
	// either a caller-supplied override or the agent's built-in default.
	Home() string
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
// interactively. Not every provider implements it — Antigravity, for
// instance, isn't driven through a resumable terminal command.
type Resumable interface {
	// ResumeCommand returns the argv (argv[0] is the executable, looked up
	// on PATH) and working directory to run in order to resume s
	// interactively, attached to the current terminal. err is
	// ErrResumeUnsupported (or wraps it) if this agent can't do that.
	ResumeCommand(s Session) (argv []string, dir string, err error)
}

// EnvHomeProvider is implemented by providers whose underlying CLI reads an
// environment variable to relocate its entire home directory (credentials,
// sessions, everything) — e.g. Claude Code's CLAUDE_CONFIG_DIR. A caller
// resuming a session under a non-default profile should set this variable
// on the child process, or it'll resume against the CLI's own default
// directory instead of the profile actually being browsed.
type EnvHomeProvider interface {
	// HomeEnv returns the environment variable name and the value it
	// should be set to for this provider instance's own home directory.
	HomeEnv() (name, value string)
}

// NewSessionStarter is implemented by providers whose CLI can start a brand
// new session directly, optionally given a name up front. Where the CLI has
// a real flag for this (Claude Code's --name), that's a clean rename, never
// sent to the model. Where it doesn't, the name is passed as the session's
// initial message instead — not a true rename, but it still ends up as the
// session's title via that provider's own title-extraction fallback.
type NewSessionStarter interface {
	// NewSession returns the argv (argv[0] looked up on PATH) and working
	// directory to start a new interactive session, named name if it's
	// non-empty, or with no name/initial message at all if it's empty.
	NewSession(name string) (argv []string, dir string, err error)
	// NameIsRename reports whether the name given to NewSession becomes a
	// real session title (true, e.g. Claude Code's --name) or is only sent
	// to the agent as its first message because the CLI has no rename flag
	// (false) — the UI needs this to label the name prompt accurately
	// instead of implying every agent can be cleanly renamed.
	NameIsRename() bool
}
