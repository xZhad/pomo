# pomo

A terminal-native pomodoro timer with session logging, mid-session notes, OS notifications, and an animated TUI. Single binary, local-first — your focus log lives in a plain JSONL file you own.

Built on [jsonldb](https://github.com/xZhad/jsonldb) for storage (and its query/aggregation engine) and [bubbletea v2](https://charm.land) + [harmonica](https://github.com/charmbracelet/harmonica) for the animated TUI.

---

## Install

Homebrew:

```sh
brew install xZhad/tap/pomo
```

Or with Go (requires Go 1.26+):

```sh
go install github.com/xZhad/pomo@latest
```

macOS/Linux (the background watcher uses `setsid`).

---

## Quick start

```sh
pomo start "axiom ch3" --work 25 -t study   # begin a 25-min session, tagged
pomo note "understood vanishing gradient"   # jot a timestamped note, any time
pomo status                                  # time remaining
pomo done                                    # end + log it
pomo last                                    # "where did I leave off?" — topic + notes
```

Running `pomo` with no arguments (or `pomo tui`) opens the **animated TUI** — a live countdown card with a springy progress bar, a breathing 🍅, inline note entry, and a history pane (`tab`).

---

## How it works

- `pomo start` writes the session to `~/.pomo/sessions.jsonl` immediately and spawns a detached background watcher that fires an OS notification at the deadline — no terminal needs to stay open.
- The active session is tracked in `~/.pomo/current.json` (id, deadline, pause state) — so `note`/`status`/`pause` work from any shell, and a crash mid-session is recoverable.
- The log is the product: every report reads from `sessions.jsonl`. It's plain JSONL — `jq`/`grep`-friendly, and any tool can read it.

Data dir defaults to `~/.pomo/` (override with `POMO_DIR`). Config in `~/.pomo/config.json`.

---

## Commands

### Timer
| Command | Action |
|---------|--------|
| `pomo start "<topic>" [--work N] [-t tag]...` | start a session (default 25 min) |
| `pomo status` | time remaining in the current session |
| `pomo note "<text>"` | add a timestamped note to the active session |
| `pomo pause` / `pomo resume` | pause / resume the countdown |
| `pomo extend [minutes]` | add time to the active session (default +5) |
| `pomo done` | end early, mark completed, show summary |
| `pomo stop` | abort without marking completed |

### History & reports
| Command | Action |
|---------|--------|
| `pomo last` | last session: topic + all notes |
| `pomo log [filter]` | session log, newest first; optional jsonldb DSL filter |
| `pomo topics` | time grouped by topic |
| `pomo report --by topic\|tag\|day` | aggregate time by topic, tag, or day |
| `pomo rm <id>` | delete a session |

`log`'s optional filter is the jsonldb query DSL, e.g. `pomo log "completed=true topic~=ml"`.

### Config
```sh
pomo config                  # show current config
pomo config set work 25      # default work minutes
pomo config set break 5      # default break minutes
pomo config set goal 4       # daily goal
pomo config set notify false # disable OS notifications
```

---

## TUI keys

`p`/`space` pause·resume · `n` note · `e` +5 min · `d` done · `s` stop · `tab` history · (history) `j`/`k` scroll · `q` quit.

---

## Data model

One session per line in `~/.pomo/sessions.jsonl`:

```json
{"id":"01J…","topic":"axiom ch3","duration":1500,"started":"2026-06-22T17:20:23-04:00","ended":"2026-06-22T17:45:23-04:00","completed":true,"tags":["study"],"notes":[{"at":"2026-06-22T17:28:00-04:00","text":"understood vanishing gradient"}]}
```

Timestamps are local (so day-bucketed reports match your calendar). Stored via `jsonldb.Typed[Session]`; reports use jsonldb's `GroupBy`/`GroupByFunc`/`Sum`.

---

## Tech

- **Storage:** [jsonldb](https://github.com/xZhad/jsonldb) — JSONL, no SQLite
- **TUI:** bubbletea v2 + lipgloss v2 + bubbles v2 + harmonica (spring animation)
- **Notifications:** [beeep](https://github.com/gen2brain/beeep)
- **IDs:** [ulid](https://github.com/oklog/ulid)

---

## Roadmap (not yet built)

v1 ships the core loop, history/reports, config, daemon notifications, and the animated TUI. Planned for later: a gamified dashboard (`dash`/`streak`/`stats`, XP, weekly heatmap), auto-scaled breaks + `skip`, `status --watch`, weekly/monthly markdown reports, and CSV/JSON `export`.

---

## License

MIT
