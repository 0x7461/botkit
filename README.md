# botkit

A lightweight Go framework for scheduled Telegram bots. Implement three interfaces — Source, Formatter, Sender — wire them together, schedule with snooze + runit.

## Framework

```go
type Source interface {
    Fetch() ([]Item, error)
}

type Formatter interface {
    Format(items []Item) string
}

type Sender interface {
    Send(message string) error
}
```

Adding a new bot = implement `Source`, pick a `Formatter` and `Sender`, pass to `bot.Bot{}`.

## Bots

- **rss-bot** — RSS feed aggregator. HN Best, Lobsters, Techmeme, blogs. SQLite dedup. Runs twice daily.
- **gh-bot** — GitHub trending repos (weekly). Scrapes trending page via goquery. Runs weekly.
- **nagger** — daily Claude-quota pace nudge. One-shot, hourly 08–22, dedup'd to one message/day.

## Project Structure

```
bot/                        — framework: interfaces + Bot runner
cmd/{rss-bot,gh-bot,nagger}/ — bot entry points
sources/{rss,github}/       — Source implementations
formatters/{rss,markdown}/  — Formatter implementations
senders/telegram/           — Telegram sender (HTML, message splitting)
```

## Configuration

Copy `.env.example` to `.env` and fill in bot tokens and chat IDs. Each bot can have its own token (`BOT_RSS__TOKEN`, `BOT_GH__TOKEN`) or fall back to `TELEGRAM_BOT_TOKEN`.

## Dependencies

- [goquery](https://github.com/PuerkitoBio/goquery) — HTML parsing (GitHub trending)
- [gofeed](https://github.com/mmcdole/gofeed) — RSS/Atom feed parsing
- [godotenv](https://github.com/joho/godotenv) — .env loading
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — SQLite (RSS dedup)
