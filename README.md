# botkit

A lightweight Go framework for scheduled Telegram bots. Implement three interfaces — Source, Formatter, Sender — wire them together, schedule with snooze + runit.

## Framework

```go
// Source fetches items from anywhere.
type Source interface {
    Fetch() ([]Item, error)
}

// Formatter turns items into a message string.
type Formatter interface {
    Format(items []Item) string
}

// Sender delivers the message somewhere.
type Sender interface {
    Send(message string) error
}
```

Adding a new bot = implement `Source`, pick a `Formatter` and `Sender`, pass to `bot.Bot{}`.

## Project Structure

```
bot/                        — framework: interfaces + Bot runner
sources/github/             — GitHubTrendingSource (first implementation)
telegram/
  sender.go                 — TelegramSender
  formatter.go              — TelegramFormatter
main.go                     — wires the GitHub trending bot together
```

## Configuration

Copy `.env.example` to `.env` and fill in:

```env
ENABLE_TELEGRAM=true
TELEGRAM_BOT_TOKEN=your-bot-token
TELEGRAM_CHAT_ID=your-chat-id
```

Get your Chat ID: send `/start` to your bot, then run `telegram.GetChatID(token)`.

## Usage

```bash
# Dry run (Telegram disabled)
go run main.go

# Build and run
go build -o botkit && ./botkit
```

## Scheduling (Void Linux / runit + snooze)

`~/service/botkit/run`:
```sh
#!/bin/sh
exec 2>&1
exec snooze -w6 -H10 -M0 /home/ta/projects/botkit/botkit
```

`~/service/botkit/log/run`:
```sh
#!/bin/sh
exec svlogd -tt ./main
```

```bash
chmod +x ~/service/botkit/run ~/service/botkit/log/run
sv status ~/service/botkit
```

Runs every Saturday at 10:00.

## Dependencies

- [goquery](https://github.com/PuerkitoBio/goquery) — HTML parsing
- [godotenv](https://github.com/joho/godotenv) — .env loading
