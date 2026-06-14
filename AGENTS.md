# AGENTS.md — botkit

Updated: 2026-06-14

Lightweight Go framework for scheduled Telegram bots. Three interfaces (Source / Formatter / Sender) wired into one runner; each bot is its own binary on a runit + snooze schedule. Three binaries ship: rss-bot (RSS digest), gh-bot (GitHub trending), and nagger (one-shot daily Claude-quota pace nudge — uses `senders/telegram` directly rather than the Source/Formatter runner).

> The `ai-agent` bot ("The Smartass" — interactive multi-backend Telegram chat) was **retired 2026-06-14** (superseded by Claude's remote-control; barely used). See `PLAN.md ## History`.

Audience: agents editing this repo. Framework overview + bot list in `README.md`. Decisions, internals, history in `PLAN.md` (local-only — gitignored).

## Setup

```bash
go mod download
cp .env.example .env   # umbrella (shared): ENABLE_TELEGRAM=true, TELEGRAM_*, GMAIL_*
# per-bot secrets go in .env.<name> (.env.gh, .env.rss, .env.nagger):
#   BOT_<NAME>__TOKEN / BOT_<NAME>__CHAT
```

Each bot loads `.env.<name>` then the umbrella `.env` via `bot.LoadEnv(name)` (first-wins: per-bot overrides shared). All `.env*` except `.env.example` are gitignored.

## Services

Declared runtime state — reconciled against `sv status` + `down` sentinels by `maint-watch doctor` and the weekly maint-watch scan (`service.claim_*` findings). `persistent` = must be up and survive reboot.

- `github-trending`: persistent — weekly Sat 10:00 gh-bot (`snooze -w6 -H10`)
- `rss-bot`: persistent — daily 12:00 RSS digest (`snooze -H12`)
- `nagger`: persistent — hourly 08–22 Claude-quota pace nudge (`snooze -H8-22 ./bin/nagger`), dedup'd to one msg/day

## Commands

```bash
# Build (run after every code change before sv restart)
go build -o bin/gh-bot   ./cmd/gh-bot/
go build -o bin/rss-bot  ./cmd/rss-bot/

# Service control
SVDIR=~/service sv status github-trending
SVDIR=~/service sv restart rss-bot

# Dry run (without ENABLE_TELEGRAM=true, bots log instead of POSTing)
go run ./cmd/rss-bot/
```

## Project layout

```
bot/                              framework: Item, Source/Formatter/Sender, Bot runner
bot/curate/                       rss-bot LLM ranking (claude -p + Ollama backends, ChainCurator)
cmd/{rss-bot,gh-bot,nagger}/      bot entry points — one binary each
sources/{rss,github}/             Source implementations (gofeed, goquery)
formatters/{rss,markdown}/        Formatter implementations
senders/telegram/                 Telegram sender + GetChatID helper
config/                           per-bot JSON config loader (~/.config/botkit/<bot>.json)
compress/                         summarize CLI shell-out (gh-bot trending summaries)
bin/                              built binaries (gitignored)
```

External integration points:
- `~/service/{github-trending,rss-bot,nagger}/` — runit user services.
- `~/.local/share/botkit/rss-seen.db` — RSS dedup SQLite.
- `~/.config/botkit/<bot>.json` — per-bot config overrides (period, summarize, feed list, max_delivery, rss-bot `curate` block).
- `~/.config/botkit/nagger.json` — nagger schedule config, read by `cmd/nagger`. Hand-edited (was written by ai-agent's `/nagger` before that bot's retirement).
- `~/.local/share/nagger/{rate-limits.json,last-sent}` — nagger pace cache (`rate-limits.json` written by `~/.claude/statusline.sh` every CC response — external, don't move) + daily dedup state.

## Boundaries & gotchas

**Always do:**
- **Set `ENABLE_TELEGRAM=true` in `.env`** for real sends; otherwise bots dry-run (log instead of POST). Required in prod.
- **`cd /path/to/project` before `exec` in runit `run` scripts.** runit doesn't set CWD; `godotenv.Load()` won't find `.env` without it.
- **Rebuild AND restart after code changes:** `go build -o bin/<bot> ./cmd/<bot>/` AND `SVDIR=~/service sv restart <bot>`. runit runs the pre-built binary from `bin/`, not `go run`. Stale binaries silently serve old behavior — hit production 2026-03-13 (formatter rewritten 2026-03-09, binary still from 2026-03-07).
- **One binary per bot.** Different schedules, different tokens, different lifecycles. Don't bundle.
- **Use `BOT_<NAME>__TOKEN` / `BOT_<NAME>__CHAT`** for per-bot Telegram credentials; falls back to generic `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` if unset. Each bot ideally has its own BotFather token. Per-bot secrets live in `.env.<name>`, shared values in the umbrella `.env`; both loaded via `bot.LoadEnv(name)`.

**Never do:**
- **Don't merge bot binaries.** Separate concerns (schedule, token, .env scope, lifecycle).
- **Don't use `cmd.Output()` for `claude -p` invocations.** When CC quota expires, `claude -p` writes the error to **stdout, not stderr**. `cmd.Output()` discards stdout on error → empty error message. Use an explicit `bytes.Buffer` for stderr and fall back to stdout content if stderr is empty. Applies to any `claude -p` shell-out (e.g. `bot/curate/`).
- **Don't bundle gh-bot's RSS feeds with rss-bot.** gh-bot is GitHub-trending-only; rss-bot is RSS-only. They run on different schedules and use different formatters.
- **Don't pass `--bare` to `claude -p`** in `bot/curate/`. `--bare` skips not just CLAUDE.md/settings but also auth discovery → "Not logged in" failure. Caught 2026-05-25 when first wiring rss-bot curation.

**Ask first:**
- Adding a new bot binary. Comes with runit service setup, BotFather token, schedule decision — discuss in PLAN.md `## Decisions` first.

**Untested / known-fragile:**
- Recovery path when `~/.config/botkit/nagger.json` is malformed. `cmd/nagger` reads it; error paths on a bad hand-edit haven't been exercised.

## Where to look

- **`README.md`** — framework overview, bot list, interface signatures, dependencies.
- **`PLAN.md ## Decisions`** (local-only) — architectural choices: separate binaries, snooze+runit scheduling, three-interface split.
- **`PLAN.md ## Internals`** — recap integration, dotenv/runit interaction details.
- **`PLAN.md ## History`** — what shipped when, including the 2026-06-14 ai-agent retirement and the 2026-03-13 Opus code-review hardening (20 issues fixed in one commit, `b10018a`).
- **`cmd/nagger/`** — quota-pace nudge, folded in 2026-06-05 (was the standalone `~/projects/nagger` Python project). Reads `~/.config/botkit/nagger.json`.
