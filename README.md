# triAGe

*(the "AG" is short for "agent")*

A terminal GUI for browsing, resuming, and cleaning up local session history
left behind by AI coding agent CLIs. Pick an agent, see every session it has
stored on this machine, resume one to jump back into it, or mark and delete
the ones you don't want.

## Install

### Prebuilt binary
**Download a prebuilt binary** from the [releases page](https://github.com/kbaljak/triAGe/releases). 

Pick the archive matching your OS/arch, extract it,
and install **triage** in *~/.local/bin/*


(Swap `linux_amd64` for `linux_arm64`, `darwin_amd64`, or `darwin_arm64` according to your needs.)

```sh
curl -LO https://github.com/kbaljak/triAGe/releases/latest/download/triage_linux_amd64.tar.gz
tar -xzf triage_linux_amd64.tar.gz
install -Dm755 triage ~/.local/bin/triage
```
### Using Go
```sh
go install github.com/kbaljak/triAGe/cmd/triage@latest
```

This installs `triage` binary to `~/go/bin`. Make sure that directory is on your `PATH`, then run **`triage`**.

### From source using make
```sh
git clone https://github.com/kbaljak/triAGe.git
cd triAGe
make build     # -> ./triage
make install   # installs to ~/.local/bin (override with PREFIX=...)
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

## Supported agents

Claude Code, Antigravity, and Codex are verified against real session data
and a real install. The rest are implemented from each project's official
docs (linked below) rather than guesswork, but are unverified end-to-end —
if you run one of these and something's off, please open an issue with
what you saw.

| Agent | Where it looks |
|---|---|
| **Claude Code** | `~/.claude/projects/*/*.jsonl` |
| **Antigravity** |`~/.gemini/antigravity-cli/{cache,conversations,annotations,presence,brain}` |
| **Codex (OpenAI)** |`~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`, names from `~/.codex/session_index.jsonl` |
| **Gemini CLI** |`~/.gemini/tmp/<project-hash>/chats/*.json` |
| **GitHub Copilot CLI** |`~/.copilot/session-state/<id>/` |
| **Grok Build (xAI)** |`~/.grok/sessions/<encoded-cwd>/<id>/summary.json` |

## Adding another agent

Implement `session.Provider` (see `internal/session/session.go`) in a new
file under `internal/session/`, then add it to `session.All()` in
`internal/session/registry.go`. Also implement `session.Resumable` on it if
the agent has a real "resume by session ID" CLI command. No change needed to `internal/ui`

## Development

```sh
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

See [CONTRIBUTING](CONTRIBUTING.md) for the full workflow, step by step.

## License

[MIT](LICENSE)
