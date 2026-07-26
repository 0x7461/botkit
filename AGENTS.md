# AGENTS.md — botkit

Updated: 2026-07-26

Lightweight Go framework for scheduled Telegram bots. Three interfaces (Source / Formatter / Sender) wired into one runner; each bot is its own binary on a runit + snooze schedule. Binaries: rss-bot (RSS digest), scout (combined GitHub trending + HN Ask/Show/Tell — `bot.MultiSource`), and nagger (daily Claude-quota pace nudge + recurring manual-task reminders — uses `senders/telegram` directly rather than the Source/Formatter runner).

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

- `scout`: persistent — weekly Sat 09:00 GitHub trending + HN digest (`snooze -w6 -H9`). Credentials in `.env.scout` (`BOT_SCOUT__TOKEN`/`BOT_SCOUT__CHAT`). Replaced `github-trending`/gh-bot (retired 2026-06-17).
- `rss-bot`: persistent — daily 12:00 RSS digest (`snooze -H12`)
- `nagger`: persistent — hourly 08–22 (`snooze -H8-22 ./bin/nagger`). Runs a unified set of periodic **nags** (`cmd/nagger`: `Nag` = id + interval + anchor + renderer): the daily Claude-quota pace check (interval 1, live-computed message) and recurring manual-task **reminders** (fixed-day cadence, e.g. quarterly archive chores; configured under `reminders` in `nagger.json`). One run evaluates all nags, fires the due ones as **one combined message** (grouped under headers), and dedups per-nag via `state.json`. Per-nag dedup makes the hourly poll idempotent (each nag fires once per cycle).

## Commands

```bash
# Build (run after every code change before sv restart)
go build -o bin/scout    ./cmd/scout/
go build -o bin/rss-bot  ./cmd/rss-bot/

# Dry-run scout without sending (umbrella .env sets ENABLE_TELEGRAM=true, so override):
ENABLE_TELEGRAM=false go run ./cmd/scout/

# Service control
SVDIR=~/service sv status scout
SVDIR=~/service sv restart rss-bot

# Dry run (without ENABLE_TELEGRAM=true, bots log instead of POSTing)
go run ./cmd/rss-bot/
```

## Project layout

```
bot/                              framework: Item, Source/Formatter/Sender, Bot runner, MultiSource
bot/curate/                       LLM passes: ranking (ChainCurator, rss-bot) + Annotate (summary+sentiment, scout), claude -p + Ollama
cmd/{rss-bot,scout,nagger}/       bot entry points — one binary each
sources/{rss,github,hackernews}/  Source implementations (gofeed, goquery, Algolia HN API)
formatters/{rss,scout}/           Formatter implementations
senders/telegram/                 Telegram sender + GetChatID helper
config/                           per-bot JSON config loader (~/.config/botkit/<bot>.json)
compress/                         summarize CLI shell-out (scout's GitHub trending summaries)
bin/                              built binaries (gitignored)
```

External integration points:
- `~/service/{scout,rss-bot,nagger}/` — runit user services.
- `~/.local/share/botkit/rss-seen.db` — RSS dedup SQLite.
- `~/.config/botkit/<bot>.json` — per-bot config overrides (scout: period/summarize/limit + hn block; rss-bot: feed list, max_delivery, `curate` block, `summarize` block — per-item one-line summaries, sonnet, off unless `enabled`). **`rss-bot.json` is chezmoi-managed** — `chezmoi re-add ~/.config/botkit/rss-bot.json` after editing it live, or the source drifts (scout/nagger json are not tracked).
- `~/.config/botkit/nagger.json` — nagger schedule config + `reminders` array, read by `cmd/nagger`. Hand-edited (was written by ai-agent's `/nagger` before that bot's retirement). Each reminder: `{id, message, every_days, anchor}` (anchor = first due date when never fired).
- `~/.local/share/nagger/{rate-limits.json,state.json}` — `rate-limits.json` is the pace cache (written by `~/.claude/statusline.sh` every CC response — external, don't move). `state.json` is the unified per-nag last-fired map (id→YYYY-MM-DD, incl. `quota`; config is immutable, state is separate). Replaced the split `last-sent` + `reminders-state.json` (2026-07-26; `readState` migrates them once if `state.json` is absent).

## Boundaries & gotchas

**Always do:**
- **Set `ENABLE_TELEGRAM=true` in `.env`** for real sends; otherwise bots dry-run (log instead of POST). Required in prod.
- **`cd /path/to/project` before `exec` in runit `run` scripts.** runit doesn't set CWD; `godotenv.Load()` won't find `.env` without it.
- **Rebuild AND restart after code changes:** `go build -o bin/<bot> ./cmd/<bot>/` AND `SVDIR=~/service sv restart <bot>`. runit runs the pre-built binary from `bin/`, not `go run`. Stale binaries silently serve old behavior — hit production 2026-03-13 (formatter rewritten 2026-03-09, binary still from 2026-03-07).
- **One binary per bot.** Different schedules, different tokens, different lifecycles. Don't bundle.
- **Use `BOT_<NAME>__TOKEN` / `BOT_<NAME>__CHAT`** for per-bot Telegram credentials; falls back to generic `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` if unset. Each bot ideally has its own BotFather token. Per-bot secrets live in `.env.<name>`, shared values in the umbrella `.env`; both loaded via `bot.LoadEnv(name)`.

**Never do:**
- **Don't merge bot binaries** *when they differ in schedule, token, .env scope, or lifecycle.* That's the test, not "always separate". Bots that share all four belong together via `bot.MultiSource` (one binary, many sources) — e.g. scout folds GitHub trending + HN into one weekly Sat digest. Don't split a single coherent digest into N binaries, and don't bundle bots with divergent cadence/tokens.
- **Don't use `cmd.Output()` for `claude -p` invocations.** When CC quota expires, `claude -p` writes the error to **stdout, not stderr**. `cmd.Output()` discards stdout on error → empty error message. Use an explicit `bytes.Buffer` for stderr and fall back to stdout content if stderr is empty. Applies to any `claude -p` shell-out (e.g. `bot/curate/`).
- **Keep GitHub trending (scout's `sources/github`) and RSS (rss-bot) as separate sources on separate bots.** Don't fold RSS feeds into scout or GitHub trending into rss-bot — different schedules, formatters, and lifecycles.
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
