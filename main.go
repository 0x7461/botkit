package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/0x7461/botkit/bot"
	github "github.com/0x7461/botkit/sources/github"
	"github.com/0x7461/botkit/telegram"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	source := &github.TrendingSource{Period: "weekly"}
	formatter := &telegram.Formatter{Title: "GitHub Trending — Weekly Report"}

	if os.Getenv("ENABLE_TELEGRAM") != "true" {
		// Dry run — fetch and print count only
		items, err := source.Fetch()
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		fmt.Printf("Found %d repos (Telegram disabled — set ENABLE_TELEGRAM=true to send)\n", len(items))
		return
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	var chatID int64
	fmt.Sscanf(os.Getenv("TELEGRAM_CHAT_ID"), "%d", &chatID)

	if token == "" || chatID == 0 {
		log.Fatal("ENABLE_TELEGRAM=true but TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is missing")
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
