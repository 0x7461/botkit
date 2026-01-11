# GitHub Trending Scraper

A lightweight Go application that scrapes GitHub's trending repositories and delivers weekly reports via Telegram.

## Overview

This project was built as a learning exercise to explore Go while creating something practical. It fetches the weekly trending repositories from GitHub, formats them nicely, and sends them to a Telegram bot. Perfect for staying up-to-date with what's hot in the developer community!

## Features

- **Web Scraping**: Fetches trending repositories from GitHub's weekly trending page
- **Telegram Integration**: Sends formatted reports to your Telegram account
- **Proper Markdown Formatting**: Handles Telegram's markdown syntax correctly
- **Automated Scheduling**: Runs weekly using snooze + runit (Voidlinux)
- **Logging**: Full logging support via svlogd

## Prerequisites

- Go 1.25.5 or higher
- Telegram account and bot token
- `snooze` package (for scheduling on Voidlinux)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/0x7461/github-trending.git
cd github-trending
```

2. Install dependencies:
```bash
go mod download
```

3. Build the executable:
```bash
go build -o github-trending
```

## Configuration

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Set up your Telegram bot:
   - Message [@BotFather](https://t.me/BotFather) on Telegram
   - Send `/newbot` and follow the prompts
   - Copy the bot token to your `.env` file

3. Get your Chat ID:
   - Send `/start` to your bot
   - Run the chat ID helper (optional, or just check the `.env` setup below)

4. Edit `.env` with your credentials:
```env
ENABLE_TELEGRAM=true
ENABLE_EMAIL=false

TELEGRAM_BOT_TOKEN=your-bot-token-here
TELEGRAM_CHAT_ID=your-chat-id-here
```

## Usage

### Manual Run

To run the scraper manually:
```bash
./github-trending
```

Or with Go:
```bash
go run main.go
```

### Automated Scheduling (Voidlinux)

This project uses `snooze` + `runit` for scheduling on Voidlinux. Here's the complete setup:

#### 1. Install snooze
```bash
sudo xbps-install -y snooze
```

#### 2. Create the user service directory structure
```bash
mkdir -p ~/service/github-trending/log/main
```

#### 3. Create the main service run script
Create `~/service/github-trending/run`:
```bash
#!/bin/sh
exec 2>&1

# Wait until Saturday at 10:00
# -w6 = Saturday (0=Sunday, 6=Saturday)
# -H10 = 10 AM
# -M0 = minute 0
exec snooze -w6 -H10 -M0 \
  /home/ta/projects/github-trending/github-trending
```

Make it executable:
```bash
chmod +x ~/service/github-trending/run
```

**Note**: Adjust the path to match your installation directory and change the schedule flags as needed.

#### 4. Create the log service run script
Create `~/service/github-trending/log/run`:
```bash
#!/bin/sh
exec svlogd -tt ./main
```

Make it executable:
```bash
chmod +x ~/service/github-trending/log/run
```

#### 5. Start the user service supervisor
Add this to your `~/.profile` to start `runsvdir` automatically on login:
```bash
# Start user service supervisor
if [ -d "$HOME/service" ] && ! pgrep -u "$(id -u)" runsvdir >/dev/null 2>&1; then
    runsvdir "$HOME/service" >/dev/null 2>&1 &
fi
```

Then source it or start manually for the current session:
```bash
runsvdir ~/service >/dev/null 2>&1 &
```

#### 6. Verify the service is running
```bash
sv status ~/service/github-trending
```

You should see something like:
```
run: /home/ta/service/github-trending: (pid 12345) 10s; run: log: (pid 12346) 10s
```

#### 7. View logs
```bash
tail -f ~/service/github-trending/log/main/current
```

**Schedule**: Runs every Saturday at 10:00 AM

**How it works**:
- `snooze` waits until the scheduled time
- Executes the scraper binary
- Sends report to Telegram
- Service automatically restarts and waits for next week
- All output is logged with timestamps

## Project Structure

```
github-trending/
├── main.go              # Main application with scraper logic
├── utils/
│   └── utils.go         # Utility functions (Telegram API, formatters)
├── .env                 # Environment configuration (not committed)
├── .env.example         # Template for environment setup
├── .gitignore           # Git ignore rules
├── go.mod               # Go module definition
├── go.sum               # Go dependencies checksums
├── LICENSE              # MIT License
└── README.md            # This file
```

**Note**: Runit service files are created separately in `~/service/github-trending/` and are not part of this repository.

## Development Journey

This project was built to learn Go from scratch. Here are some key decisions and learnings:

### Language Choice
- **Considered**: Python, Rust, Go
- **Chosen**: Go for its simplicity, great standard library, and suitability for this type of task
- **Learning Focus**: Package structure, error handling, HTTP requests, string manipulation

### Delivery Method
- **Initially planned**: Gmail SMTP
- **Final choice**: Telegram bot (primary)
- **Reasoning**: Simpler API, no OAuth complexity, integrated with existing Telegram backup workflow

### Formatting Challenges
- GitHub's trending page has unusual whitespace in repo names
- **Solution**: Used `strings.Fields()` to collapse all whitespace
- Telegram uses different markdown than standard markdown
- **Solution**: Created separate `FormatForTelegram()` function
- Telegram's legacy Markdown mode required specific character escaping
- **Solution**: Only escape `_`, `*`, `[`, `` ` `` (not all special chars like MarkdownV2)

### Architecture Decisions
- **Repo struct**: Moved to `utils` package to avoid circular dependencies
- **Formatters**: Refactored into `utils` for reusability and cleaner main.go
- **Environment flags**: Used `ENABLE_TELEGRAM` and `ENABLE_EMAIL` for flexibility

### Scheduling on Voidlinux
- **Cron not available**: Voidlinux doesn't ship with cron by default
- **Chosen solution**: `snooze` + `runit`
- **Why**: Native to Voidlinux's init system, no daemon needed, elegant design

### Go Concepts Learned
- Package imports and module paths
- Struct definitions and methods
- Multiple return values (result, error pattern)
- `defer` for cleanup
- String builders for efficient concatenation
- Anonymous structs for JSON parsing
- HTTP client usage
- goquery for HTML parsing

## Dependencies

- [goquery](https://github.com/PuerkitoBio/goquery) - HTML parsing
- [godotenv](https://github.com/joho/godotenv) - Environment variable loading

## Future Enhancements

Potential features to add:
- [ ] Email delivery via Gmail SMTP
- [ ] Filter by programming language
- [ ] Customizable schedule
- [ ] HTML formatting option
- [ ] Database to track historical trends

## License

MIT License - Feel free to use and modify as you wish!

## Acknowledgments

Built as a learning project to explore Go programming with assistance from [Claude Code](https://claude.com/claude-code). Thanks to:
- GitHub for the trending page
- Telegram for the simple bot API
- The Go community for excellent documentation
- Claude Code for guidance and pair programming

---

**Note**: This project is for educational purposes. Please be respectful of GitHub's servers and don't abuse the scraping functionality.
