# botkit

Lightweight Go framework for scheduled Telegram bots.

## Status

**Maintain** — Framework extracted, GitHub trending bot running in production.

## Architecture

```
cmd/gh-bot/main.go              — wires gh trending bot (source + formatter + sender)
bot/bot.go                      — Item struct, Source/Formatter/Sender interfaces, Bot.Run()
sources/github/trending.go      — GitHubTrendingSource (weekly, configurable period)
formatters/markdown/formatter.go — Telegram Markdown formatter
senders/telegram/sender.go      — TelegramSender + GetChatID helper
bin/                            — built binaries (gitignored), build with:
                                  go build -o bin/gh-bot ./cmd/gh-bot/
```

## Running Bot

GitHub trending → Telegram, every Saturday 10:00 via runit + snooze.

Service: `~/service/github-trending/`

## Adding a New Bot

1. Implement `Source` in `sources/<name>/`
2. Reuse `telegram.Formatter` + `telegram.Sender` or implement custom ones
3. Wire in a new `main_<name>.go` or add a subcommand to `main.go`

## Roadmap

- [ ] Config file for source/formatter/sender selection (no recompile needed)

---

## RSS Bot Plan

### Goal

New bot: fetch curated RSS feeds, deduplicate seen items, deliver new ones to Telegram. Scheduled every 6h via runit + snooze.

### New files

```
sources/rss/
  source.go     — RSSSource, FeedConfig, Fetch()
  dedup.go      — SQLite seen-items tracker
formatters/rss/
  formatter.go  — groups items by feed, Telegram Markdown output
cmd/rss-bot/
  main.go       — wire feeds + dedup + formatter + sender
```

### Phase 1: RSS Source (`sources/rss/source.go`)

```go
type FeedConfig struct {
    Name     string
    URL      string
    MaxItems int // per fetch, default 5
}

type RSSSource struct {
    Feeds []FeedConfig
}

func (s *RSSSource) Fetch() ([]bot.Item, error)
```

- Use `github.com/mmcdole/gofeed` to parse RSS/Atom
- Item.Meta: `"feed"` = feed name, `"guid"` = item GUID or URL (for dedup)
- Cap per-feed items to MaxItems (newest first)

### Phase 2: Deduplication (`sources/rss/dedup.go`)

SQLite DB at `~/.local/share/botkit/rss-seen.db`.

```go
type Deduplicator struct{ db *sql.DB }

func NewDeduplicator(path string) (*Deduplicator, error)  // creates table if not exists
func (d *Deduplicator) Filter(items []bot.Item) []bot.Item // returns only unseen items
func (d *Deduplicator) MarkSeen(items []bot.Item) error    // records items after delivery
```

Schema: `CREATE TABLE seen (guid TEXT PRIMARY KEY, feed TEXT, seen_at INTEGER)`

Use `modernc.org/sqlite` — pure Go, no cgo, no system lib needed.

### Phase 3: Formatter (`formatters/rss/formatter.go`)

Groups items by feed name, outputs Markdown:

```
📰 *HN Best*
• [Title](url)
• [Title](url)

📰 *Lobsters*
• [Title](url)
```

### Phase 4: `cmd/rss-bot/main.go`

Wire: `RSSSource` → `Deduplicator.Filter` → `RSSFormatter` → `TelegramSender` → `Deduplicator.MarkSeen`

Initial feed list (from IDEAS.md):
- `hnrss.org/best` — HN filtered
- `lobste.rs/rss` — dev/systems community
- `techmeme.com/feed.xml` — curated tech news
- `danluu.com/atom.xml` — engineering essays
- `jvns.ca/atom.xml` — Linux/networking

### Phase 5: runit service

`~/service/rss-bot/run` — `snooze -H 6 rss-bot` (every 6h).

### Dependencies to add

```sh
go get github.com/mmcdole/gofeed
go get modernc.org/sqlite
```

### Delivery design

- If 0 new items: send nothing (skip Telegram)
- If >20 new items (first run / after outage): cap at 20 to avoid flood
- Items delivered newest-first within each feed
- MarkSeen called only after successful Send — no lost items on failure

## Notes

- Each source should be a separate bot/binary with its own schedule and message format — don't bundle sources into one delivery. Multiple bots can share the same Telegram bot token but deliver independently.

---

**Created:** 2026-02-22
