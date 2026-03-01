# botkit

Lightweight Go framework for scheduled Telegram bots.

## Status

**Maintain** — Framework extracted, GitHub trending bot running in production.

## Architecture

```
bot/bot.go                  — Item struct, Source/Formatter/Sender interfaces, Bot.Run()
sources/github/trending.go  — GitHubTrendingSource (weekly, configurable period)
telegram/sender.go          — TelegramSender + GetChatID helper
telegram/formatter.go       — TelegramFormatter (legacy Markdown)
main.go                     — wires GitHub trending bot together
```

## Running Bot

GitHub trending → Telegram, every Saturday 10:00 via runit + snooze.

Service: `~/service/botkit/`

## Adding a New Bot

1. Implement `Source` in `sources/<name>/`
2. Reuse `telegram.Formatter` + `telegram.Sender` or implement custom ones
3. Wire in a new `main_<name>.go` or add a subcommand to `main.go`

## Roadmap

- [ ] Config file for source/formatter/sender selection (no recompile needed)

## Notes

- Each source should be a separate bot/binary with its own schedule and message format — don't bundle sources into one delivery. Multiple bots can share the same Telegram bot token but deliver independently.

---

**Created:** 2026-02-22
