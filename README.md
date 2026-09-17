# triAGe

*(the "AG" is capitalized on purpose — it's short for "agent")*

A terminal GUI for browsing, resuming, and cleaning up local session history
left behind by AI coding agent CLIs. Pick an agent, see every session it has
stored on this machine, resume one to jump back into it, or mark and delete
the ones you don't want.

## Install

**Go install (fastest, needs a [Go](https://go.dev/dl/) toolchain):**

```sh
go install github.com/kbaljak/triAGe/cmd/triage@latest
```

This installs a `triage` binary to `$(go env GOPATH)/bin` (usually
`~/go/bin`). Make sure that directory is on your `PATH`, then run:

```sh
triage
```

**From source, with `make`:**

```sh
git clone git@github.com:kbaljak/triAGe.git
cd triAGe
make build     # -> ./triage
make install   # installs to ~/.local/bin (override with PREFIX=...)
```

`make` not installed? `sudo apt install make` (Debian/Ubuntu),
`brew install make` (macOS), etc. — or use the zero-dependency fallback
below.

**From source, without `make`:**

```sh
git clone git@github.com:kbaljak/triAGe.git
cd triAGe
./build.sh          # -> ./triage
./build.sh run       # build + run immediately
```

**Plain `go build`, either way:**

```sh
go build -o triage ./cmd/triage
./triage
```

## Keybindings

**Agent list**
| Key | Action |
|---|---|
| `↑`/`↓`, `j`/`k` | Move |
| `enter` | Open the selected agent's sessions |
| `r` | Re-scan every agent |
| `/` | Filter agents |
| `q`, `ctrl+c` | Quit |

**Session list**
| Key | Action |
|---|---|
| `↑`/`↓`, `j`/`k` | Move |
| `enter` | Resume the highlighted session — hands the terminal to the agent's own CLI |
| `space` | Mark/unmark the highlighted session |
| `d`, `delete` | Delete — the marked sessions, or the highlighted one if none are marked |
| `r` | Re-scan this agent's sessions |
| `esc`, `backspace`, `←` | Back to the agent list |
| `/` | Filter sessions (matches title and project) |
| `q`, `ctrl+c` | Quit |

**Delete confirmation**
| Key | Action |
|---|---|
| `y`, `enter` | Confirm — permanently deletes the file(s), cannot be undone |
| `n`, `esc` | Cancel |

## Resuming a session

`enter` on a session shells out to that agent's own CLI with whatever flag
resumes a session by ID, run from the session's original working directory,
via Bubble Tea's `tea.ExecProcess` (the same suspend-terminal/run/restore
mechanism apps use to shell out to `$EDITOR`). triAGe's UI is gone from the
screen for as long as that CLI is running — you're driving the real agent —
and triAGe reappears (with that agent's session list refreshed) once you
exit it.

Not every agent supports this yet; see the table below. Pressing `enter` on
one that doesn't shows a status message rather than doing nothing silently.

## Supported agents

Session storage formats aren't documented anywhere official, so support
varies by how much of each was actually inspected on a real machine:

| Agent | Sessions | Resume | Where it looks |
|---|---|---|---|
| **Claude Code** | Verified against real data | `claude --resume <id>`, confirmed via `claude --help` | `~/.claude/projects/*/*.jsonl` |
| **Antigravity** | Verified against real data | Not supported — no known CLI-resumable command | `~/.gemini/antigravity-cli/{cache,conversations,annotations,presence,brain}` |
| **Gemini CLI** | Best-effort, unverified (not installed on the dev machine) | Not supported | `~/.gemini/tmp/*/chats/*.json` |
| **Codex (OpenAI)** | Best-effort, unverified (not installed on the dev machine) | Best-effort: `codex resume <id>`, also unverified | `~/.codex/sessions/**/*.jsonl` |
| **GitHub Copilot CLI** | Detected, but exposes no local per-conversation history to list (only process logs and IDE lock files) | N/A (no sessions to resume) | `~/.copilot` |

There's no "ChatGPT" entry: the ChatGPT web/desktop apps don't keep locally
readable session files. The closest real equivalent — an OpenAI terminal
coding agent with on-disk sessions — is Codex CLI, listed above.

The "best-effort" providers degrade gracefully: if their assumed file format
turns out to be wrong, they just find zero sessions (or fall back to
filename/timestamp for the title) rather than crashing. Delete is always
`os.RemoveAll` over whatever paths that provider recorded for the session,
scoped to that agent's own data directory.

## Project layout

```
cmd/triage/main.go     — entry point (kept in its own dir so `go install`
                          names the binary "triage" regardless of the
                          module's own path casing)
internal/session/       — one file per agent: discovers, lists, resumes,
                          and deletes its sessions on disk
internal/ui/            — the Bubble Tea terminal GUI
```

## Adding another agent

Implement `session.Provider` (see `internal/session/session.go`) in a new
file under `internal/session/`, then add it to `session.All()` in
`internal/session/registry.go`. Also implement `session.Resumable` on it if
the agent has a real "resume by session ID" CLI command. Nothing in
`internal/ui` needs to change either way.

## Development

```sh
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

## License

[MIT](LICENSE)
