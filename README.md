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
| `p` | Set (or clear) a custom config path for the selected agent |
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
| `f` | Advanced filter — age and/or regex, see below |
| `r` | Re-scan this agent's sessions |
| `esc`, `backspace`, `←` | Back to the agent list |
| `/` | Quick fuzzy search (matches title and project) |
| `q`, `ctrl+c` | Quit |

**Advanced filter (`f`)**

Type a query combining an optional age constraint and an optional regex,
space-separated in either order:

- `days>N` — only sessions **older** than N days (by last-activity time)
- `days<N` — only sessions **newer** than N days
- anything else — a real (RE2) regex, case-insensitive, matched against the
  session **title only** — not the project path, so an incidental letter
  in your home directory (e.g. from your username) can't make an unrelated
  pattern match every session. Use `/` if you want to search by project.

This is regex, not a shell glob — `*` means "zero or more of the character
right before it," not "anything." `ju*` matches `j`, `ju`, `juu`, … and,
being unanchored, matches that anywhere in the title — so it'd match a bare
`j` in a title that has no "ju" in it at all. For "starts with ju, followed
by anything," use `ju.*` (the `.` means "any character," and the `*`
repeats *that*).

Examples: `days>30` · `kube.*prod` · `days<7 fix`. Leave it empty and press
enter to clear. This composes with `/`'s quick search — `/` filters further
within whatever the advanced filter already narrowed down to.

**Delete confirmation**
| Key | Action |
|---|---|
| `y`, `enter` | Confirm — permanently deletes the file(s), cannot be undone |
| `n`, `esc` | Cancel |

## Agents installed somewhere nonstandard

triAGe looks for each agent at its usual default location. If an agent
isn't found there — including if none are found at all on first run —
its row in the agent list shows "not detected" instead of a session count.
Highlight it and press `p` to type in the actual path to its config
directory (the same directory it would otherwise look for by default, e.g.
`~/.claude` for Claude Code); leaving the prompt empty and pressing enter
clears a path you'd previously set, reverting to the default. This works
for any agent, not just undetected ones — useful if you want to point at a
different install than the default.

Paths are saved to a config file (`os.UserConfigDir()/triage/config.json` —
typically `~/.config/triage/config.json` on Linux) and reused on the next
launch.

## Supported agents

Claude Code, Antigravity, and Codex are verified against real session data.
The rest are implemented from each project's official docs rather than
guesswork, but are unverified end-to-end — if you run one of these and
something's off, please open an issue with what you saw.

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
`internal/session/registry.go` (use `resolveHome` from `util.go` so it
supports a custom-path override like every other provider). Also implement
`session.Resumable` on it if the agent has a real "resume by session ID"
CLI command. No change needed to `internal/ui`.

## Development

```sh
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

## License

[MIT](LICENSE)
