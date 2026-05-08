package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/0x7461/botkit/bot"
	"github.com/0x7461/botkit/config"
	"github.com/0x7461/botkit/formatters/markdown"
	github "github.com/0x7461/botkit/sources/github"
	"github.com/0x7461/botkit/senders/telegram"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	cfg := config.GhBotConfig{}
	cfg.Source.Period = "weekly"
	cfg.Source.Summarize = true
	cfg.Formatter.Title = "GitHub Trending — Weekly Report"
	if err := config.Load("gh-bot", &cfg); err != nil {
		log.Printf("Warning: could not load gh-bot config: %v", err)
	}

	source := &github.TrendingSource{Period: cfg.Source.Period, Summarize: cfg.Source.Summarize}
	formatter := &markdown.Formatter{Title: cfg.Formatter.Title}

	if os.Getenv("ENABLE_TELEGRAM") != "true" {
		// Dry run — fetch and print count only
		items, err := source.Fetch()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Printf("Found %d repos (Telegram disabled — set ENABLE_TELEGRAM=true to send)\n", len(items))
		return
	}

	token := bot.FirstNonEmpty(os.Getenv("BOT_GH__TOKEN"), os.Getenv("TELEGRAM_BOT_TOKEN"))
	var chatID int64
	chatStr := bot.FirstNonEmpty(os.Getenv("BOT_GH__CHAT"), os.Getenv("TELEGRAM_CHAT_ID"))
	if chatStr != "" {
		if _, err := fmt.Sscanf(chatStr, "%d", &chatID); err != nil {
			log.Fatalf("invalid chat ID %q: %v", chatStr, err)
		}
	}

	if token == "" || chatID == 0 {
		log.Fatal("ENABLE_TELEGRAM=true but BOT_GH__TOKEN/BOT_GH__CHAT (or TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID) is missing")
	}

	b := &bot.Bot{
		Source:    source,
		Formatter: formatter,
		Sender:    &telegram.Sender{Token: token, ChatID: chatID},
	}

	fmt.Println("Fetching GitHub trending repos...")
	if err := b.Run(); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
	fmt.Println("[OK] Done!")
}
