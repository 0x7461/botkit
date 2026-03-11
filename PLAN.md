# botkit

Lightweight Go framework for scheduled Telegram bots.

**Status:** Maintain

---

## Architecture

```
bot/bot.go                       — Item struct, Source/Formatter/Sender interfaces
sources/github/trending.go       — GitHubTrendingSource (weekly, configurable period)
sources/rss/source.go            — RSSSource (gofeed, multi-feed, MaxItems cap)
sources/rss/dedup.go             — SQLite deduplication (~/.local/share/botkit/rss-seen.db)
formatters/markdown/formatter.go — Generic Telegram Markdown formatter
formatters/rss/formatter.go      — Groups items by feed, Telegram Markdown
senders/telegram/sender.go       — TelegramSender + GetChatID helper
cmd/gh-bot/main.go               — GitHub trending bot binary
cmd/rss-bot/main.go              — RSS digest bot binary
cmd/ai-agent/main.go             — AI chat agent ("The Smartass")
cmd/ai-agent/llm.go              — LLM interface + model registry
cmd/ai-agent/ollama.go           — Ollama backend (local models)
cmd/ai-agent/claudecode.go       — Claude Code CLI backend (Pro plan)
cmd/ai-agent/claude.go           — Claude API backend (API key)
cmd/ai-agent/telegram.go         — Long-polling + send
cmd/ai-agent/history.go          — SQLite conversation history
bin/                             — Built binaries (gitignored)
```

Build:
```sh
go build -o bin/gh-bot ./cmd/gh-bot/
go build -o bin/rss-bot ./cmd/rss-bot/
go build -o bin/ai-agent ./cmd/ai-agent/
```

---

## Running Bots

| Bot | Schedule | Service |
|-----|----------|---------|
| GitHub trending | Saturday 10:00 | `~/service/github-trending/` |
| RSS digest | Every 6h | `~/service/rss-bot/` |
| AI agent | Always-on (long-poll) | `~/service/ai-agent/` |

Both use `snooze` + runit (user services in `~/service/`). Restart with `SVDIR=~/service sv restart <name>`. RSS bot deduplicates via SQLite so it's safe to run frequently.

**RSS feeds:** HN Best, Lobsters, Techmeme, Dan Luu, Julia Evans. Capped at 20 items/run to prevent flooding after outage.

---

## Adding a New Bot

1. Implement `Source` in `sources/<name>/`
2. Reuse existing `Formatter` + `Sender` or implement custom ones
3. Wire in `cmd/<name>/main.go`
4. Build binary, add runit service

---

## Roadmap

- [ ] Config file for source/formatter/sender selection (no recompile needed)
- [ ] Attach 2-line summaries to GitHub trending entries (via compress/#32 `summarize` CLI)

---

## Notes

- Each bot is a separate binary with its own schedule — don't bundle multiple sources into one delivery
- Each bot has its own Telegram token via `BOT_<NAME>__TOKEN` / `BOT_<NAME>__CHAT`; falls back to `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` if not set
- **Future:** as more bots are added (e.g. naggers), create a new BotFather bot and add `BOT_<NAME>__TOKEN` / `BOT_<NAME>__CHAT` to `.env` — no code changes needed in existing bots
- `ENABLE_TELEGRAM=true` env var required to actually send; dry-run mode otherwise
- `runit + snooze` is the scheduler — no cron needed
- **runit + dotenv CWD**: runit doesn't set CWD — run scripts must `cd /path/to/project` before exec so `godotenv.Load()` finds `.env`

**Created:** 2026-02-22
**Updated:** 2026-03-12
