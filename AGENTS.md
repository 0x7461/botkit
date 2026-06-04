# AGENTS.md — botkit

Updated: 2026-05-30

Lightweight Go framework for scheduled Telegram bots. Three interfaces (Source / Formatter / Sender) wired into one runner; each bot is its own binary on a runit + snooze schedule. Four binaries ship: rss-bot (RSS digest), gh-bot (GitHub trending), ai-agent ("The Smartass" — multi-backend Telegram chat across Ollama / Claude Code CLI / Claude API), and nagger (one-shot daily Claude-quota pace nudge — uses `senders/telegram` directly rather than the Source/Formatter runner).

Audience: agents editing this repo. Framework overview + bot list in `README.md`. Decisions, internals, history in `PLAN.md` (local-only — gitignored).

## Setup

```bash
go mod download
cp .env.example .env   # fill BOT_*__TOKEN / BOT_*__CHAT; set ENABLE_TELEGRAM=true
```

## Services

Declared runtime state — reconciled against `sv status` + `down` sentinels by `maint-watch doctor` and the weekly maint-watch scan (`service.claim_*` findings). `persistent` = must be up and survive reboot.

- `ai-agent`: persistent — always-on agent daemon (`bin/ai-agent`)
- `github-trending`: persistent — weekly Sat 10:00 gh-bot (`snooze -w6 -H10`)
- `rss-bot`: persistent — daily 12:00 RSS digest (`snooze -H12`)
- `nagger`: persistent — hourly 08–22 Claude-quota pace nudge (`snooze -H8-22 ./bin/nagger`), dedup'd to one msg/day

## Commands

```bash
# Build (run after every code change before sv restart)
go build -o bin/gh-bot   ./cmd/gh-bot/
go build -o bin/rss-bot  ./cmd/rss-bot/
go build -o bin/ai-agent ./cmd/ai-agent/

# Service control
SVDIR=~/service sv status github-trending
SVDIR=~/service sv restart rss-bot
SVDIR=~/service sv restart ai-agent

# Dry run (without ENABLE_TELEGRAM=true, bots log instead of POSTing)
go run ./cmd/rss-bot/
```

## Project layout

```
bot/                              framework: Item, Source/Formatter/Sender, Bot runner
bot/curate/                       rss-bot LLM ranking (claude -p + Ollama backends, ChainCurator)
cmd/{rss-bot,gh-bot,ai-agent,nagger}/  bot entry points — one binary each
sources/{rss,github}/             Source implementations (gofeed, goquery)
formatters/{rss,markdown}/        Formatter implementations
senders/telegram/                 Telegram sender + GetChatID helper
config/                           per-bot JSON config loader (~/.config/botkit/<bot>.json)
compress/                         summarize CLI shell-out (gh-bot trending summaries)
bin/                              built binaries (gitignored)
```

ai-agent internals:
```
cmd/ai-agent/
├── main.go        dispatch + lifecycle
├── llm.go         LLM interface + model registry (haiku/sonnet/opus, gemma4)
├── ollama.go      local backend
├── claudecode.go  `claude -p` shell-out
├── claude.go      Claude API key backend
├── telegram.go    long-polling + send (cancelable context)
├── history.go     SQLite conversation history
└── nagger.go      /nagger command — reads/writes ~/.config/botkit/nagger.json (shared with cmd/nagger)
```

External integration points:
- `~/service/{github-trending,rss-bot,ai-agent}/` — runit user services.
- `~/.local/share/botkit/rss-seen.db` — RSS dedup SQLite.
- `~/.local/share/botkit/ai-agent.db` — chat history; `recap.py` reads it for the Telegram section.
- `~/.config/botkit/<bot>.json` — per-bot config overrides (period, summarize, feed list, max_delivery, rss-bot `curate` block).
- `~/.config/botkit/nagger.json` — shared nagger schedule config; written by `/nagger`, read by `cmd/nagger`.
- `~/.local/share/nagger/{rate-limits.json,last-sent}` — nagger pace cache (`rate-limits.json` written by `~/.claude/statusline.sh` every CC response — external, don't move) + daily dedup state.

## Boundaries & gotchas

**Always do:**
- **Set `ENABLE_TELEGRAM=true` in `.env`** for real sends; otherwise bots dry-run (log instead of POST). Required in prod.
- **`cd /path/to/project` before `exec` in runit `run` scripts.** runit doesn't set CWD; `godotenv.Load()` won't find `.env` without it.
- **Rebuild AND restart after code changes:** `go build -o bin/<bot> ./cmd/<bot>/` AND `SVDIR=~/service sv restart <bot>`. runit runs the pre-built binary from `bin/`, not `go run`. Stale binaries silently serve old behavior — hit production 2026-03-13 (formatter rewritten 2026-03-09, binary still from 2026-03-07).
- **One binary per bot.** Different schedules, different tokens, different lifecycles. Don't bundle.
- **Use `BOT_<NAME>__TOKEN` / `BOT_<NAME>__CHAT`** for per-bot Telegram credentials; falls back to generic `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` if unset. Each bot ideally has its own BotFather token.
- **Pass a cancelable `context.Context` into long-poll HTTP calls.** Signal handler cancels it; otherwise SIGTERM takes up to 30s to land (long-poll holds the process open). See `cmd/ai-agent/telegram.go`.

**Never do:**
- **Don't merge bot binaries.** Separate concerns (schedule, token, .env scope, lifecycle).
- **Don't use `cmd.Output()` for `claude -p` invocations.** When CC quota expires, `claude -p` writes the error to **stdout, not stderr**. `cmd.Output()` discards stdout on error → empty error message. Use an explicit `bytes.Buffer` for stderr and fall back to stdout content if stderr is empty. See `cmd/ai-agent/claudecode.go`.
- **Don't bundle gh-bot's RSS feeds with rss-bot.** gh-bot is GitHub-trending-only; rss-bot is RSS-only. They run on different schedules and use different formatters.
- **Don't pass `--bare` to `claude -p`** in `bot/curate/`. `--bare` skips not just CLAUDE.md/settings but also auth discovery → "Not logged in" failure. Caught 2026-05-25 when first wiring rss-bot curation.

**Ask first:**
- Adding a new bot binary. Comes with runit service setup, BotFather token, schedule decision — discuss in PLAN.md `## Decisions` first.
- Switching ai-agent default model. The model guard requires `/model <name> confirm` for sonnet/opus; agents shouldn't bypass it.

**Untested / known-fragile:**
- Recovery path when `~/.config/botkit/nagger.json` is malformed by `/nagger`. The command writes valid JSON on success, but error paths haven't been exercised.

## ai-agent backends

Three backends, switchable at runtime via `/model`:
- **ollama** — local. `gemma4:e4b` default (3.4 GB / 4 GB VRAM). Set `think: false` in API call to suppress thinking-mode output (CLI `ollama run` defaults to thinking on; bot unaffected).
- **claude-code** — wraps `claude -p`. Pro-plan path. `haiku` is default; `sonnet`/`opus` require explicit `confirm`.
- **claude** — direct Claude API key.

On any CC error (typically quota exhaustion), the bot **persistently switches to gemma4** via `history.SetModel` and notifies the user. User issues `/model haiku` after quota resets.

Commands:
- `/model [name [confirm]]` — switch backend or list
- `/clear` — wipe conversation history
- `/tools <passphrase>` — passphrase-gated file tools
- `/nagger [day hour]` — read/write companion project's `config.toml`

DB: `~/.local/share/botkit/ai-agent.db`. The `messages` table has a `model` column for per-turn attribution; `recap.py` queries it grouped by 30-min gaps.

## Where to look

- **`README.md`** — framework overview, bot list, interface signatures, dependencies.
- **`PLAN.md ## Decisions`** (local-only) — architectural choices: separate binaries, snooze+runit scheduling, three-interface split.
- **`PLAN.md ## Internals`** — model-guard rationale, recap integration, dotenv/runit interaction details.
- **`PLAN.md ## History`** — what shipped when, including 2026-03-13 Opus code-review hardening (20 issues fixed in one commit, `b10018a`).
- **`cmd/nagger/`** — quota-pace nudge, folded in 2026-06-05 (was the standalone `~/projects/nagger` Python project). Shares `~/.config/botkit/nagger.json` with ai-agent's `/nagger`.
