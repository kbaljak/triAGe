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
| `enter` | Choose which profile to open (always — "Default" is one of the choices) |
| `p` | Set (or clear) the Default profile's config path |
| `r` | Re-scan every agent |
| `/` | Filter agents |
| `q`, `ctrl+c` | Quit |

**Profile list** (shown for every agent — "Default" is always the first entry)
| Key | Action |
|---|---|
| `↑`/`↓`, `j`/`k` | Move |
| `enter` | Open that profile's sessions |
| `n` | Add a new profile |
| `p` | Set the highlighted profile's config path |
| `d`, `delete` | Delete the highlighted profile (not Default) — asks to confirm first |
| `r` | Re-scan this agent's profiles |
| `esc`, `backspace`, `←` | Back to the agent list |
| `q`, `ctrl+c` | Quit |

**Session list** (sorted newest-first by last activity)
| Key | Action |
|---|---|
| `↑`/`↓`, `j`/`k` | Move |
| `enter` | Resume the highlighted session — hands the terminal to the agent's own CLI |
| `n` | Start a brand-new session, optionally named |
| `space` | Mark/unmark the highlighted session |
| `d`, `delete` | Delete — the marked sessions, or the highlighted one if none are marked |
| `f` | Advanced filter — age and/or regex, see below |
| `r` | Re-scan this agent's sessions |
| `esc`, `backspace`, `←` | Back to the agent list |
| `/` | Quick fuzzy search (matches title and project folder name) |
| `q`, `ctrl+c` | Quit |

**Advanced filter (`f`)**

Type a query combining an optional age constraint and an optional regex,
space-separated in either order:

- `days>N` — sessions N days old or older
- `days<N` — sessions newer than N days
- anything else — a case-insensitive regex (RE2 syntax), matched against
  the session title only (use `/` to search by project instead)

Examples:

- `days>30` — sessions 30 days old or older
- `kube.*prod` — titles matching the regex `kube.*prod`
- `days<7 fix` — sessions newer than 7 days, with "fix" in the title

Leave it empty and press enter to clear. This composes with `/`'s quick
search — `/` filters further within whatever the advanced filter already
narrowed down to.

**Delete confirmation**
| Key | Action |
|---|---|
| `y`, `enter` | Confirm — permanently deletes the file(s), cannot be undone |
| `n`, `esc` | Cancel |

## Profiles

Selecting an agent always opens its profile list first. Every agent has an
implicit **Default** profile there: its usual config directory (e.g.
`~/.claude` for Claude Code), or a single override set with `p` on the
agent list. If an agent isn't found there — including if none are found at
all on first run — its row shows "not detected" instead of a session
count; `p` (on either screen) points it at the real location.

Add more named profiles (e.g. a "work" and a "personal" one, each a
separate account's config directory) with `n` on the profile list.

Entering a path that doesn't exist yet creates it; an existing path is used
as-is. All of this is saved to a config file
(`os.UserConfigDir()/triage/config.json` — typically
`~/.config/triage/config.json` on Linux) and reused on the next launch.

Creating a profile only creates an empty directory — triAGe has no way to
(and shouldn't try to) set up real credentials or config in it, since only
the actual agent CLI can do its own first-run authentication. A brand-new
profile shows zero sessions until the real CLI has actually run there once;
press `n` on its (empty) session list to do that — see "New sessions"
below.

**Resuming under — or starting a new session in — a non-default profile**
sets the real agent CLI's own config-directory environment variable before
launching it, so it runs against that profile's actual account and data —
not whichever one happens to be the CLI's own default:

| Agent | Env var |
|---|---|
| Claude Code | `CLAUDE_CONFIG_DIR` |
| Codex (OpenAI) | `CODEX_HOME` |
| Gemini CLI | `GEMINI_CLI_HOME` |
| GitHub Copilot CLI | `COPILOT_HOME` |
| Grok Build (xAI) | `GROK_HOME` |

Antigravity has no such variable and doesn't support resuming or starting
sessions from triAGe at all — profiles for it only control which directory
gets scanned for listing.

## New sessions

`n` on the session list starts a brand-new session with the active
agent/profile, optionally named. Leave it empty and press enter to start
untitled.

How the name is actually applied depends on what each agent's CLI
supports:

| Agent | Supported | How |
|---|---|---|
| Claude Code | Yes | `claude --name <name>` — a real rename, never sent to the model |
| Codex (OpenAI) | Yes, best-effort | No rename flag — sent as the session's first message instead |
| Gemini CLI | Yes, best-effort | Same — no rename flag, sent as the first message |
| Grok Build (xAI) | Yes, best-effort | Same — no rename flag, sent as the first message |
| GitHub Copilot CLI | No | Unverified syntax |
| Antigravity | No | No CLI-driven session creation |

For the three "best-effort" agents, the name isn't a true rename — it's
sent to the model as your first message, and only becomes the session's
displayed title because each provider's own title-extraction already falls
back to that first message when nothing better exists yet. The `n` prompt
itself is labeled accordingly: "optional first message" instead of
"optional session name" for these three.

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
file under `internal/session/`, then add its ID to `session.IDs` in
`internal/session/registry.go` (use `resolveHome` from `util.go` so it
supports a custom-path override like every other provider, and implement
`Home()` to expose it). Also implement, if the agent's CLI supports them:
`session.Resumable` (a real "resume by session ID" command),
`session.EnvHomeProvider` (an environment variable that relocates its
config directory, so profiles resume/start correctly), and
`session.NewSessionStarter` (a way to start a session non-interactively
seeded with an initial prompt). No change needed to `internal/ui`.

## Development

```sh
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

## License

[MIT](LICENSE)
