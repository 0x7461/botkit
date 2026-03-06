package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	rssformatter "github.com/0x7461/botkit/formatters/rss"
	"github.com/0x7461/botkit/senders/telegram"
	"github.com/0x7461/botkit/sources/rss"
)

const maxDelivery = 20 // cap items per run to avoid flooding

var feeds = []rss.FeedConfig{
	{Name: "HN Best", URL: "https://hnrss.org/best", MaxItems: 5},
	{Name: "Lobsters", URL: "https://lobste.rs/rss", MaxItems: 5},
	{Name: "Techmeme", URL: "https://techmeme.com/feed.xml", MaxItems: 5},
	{Name: "Dan Luu", URL: "https://danluu.com/atom.xml", MaxItems: 3},
	{Name: "Julia Evans", URL: "https://jvns.ca/atom.xml", MaxItems: 3},
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	// Deduplication DB
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".local", "share", "botkit", "rss-seen.db")
	dedup, err := rss.NewDeduplicator(dbPath)
	if err != nil {
		log.Fatalf("dedup init: %v", err)
	}
	defer dedup.Close()

	// Fetch
	source := &rss.RSSSource{Feeds: feeds}
	items, err := source.Fetch()
	if err != nil {
		log.Fatalf("fetch: %v", err)
	}

	// Filter seen
	unseen, err := dedup.Filter(items)
	if err != nil {
		log.Fatalf("dedup filter: %v", err)
	}

	// Cap to avoid flooding after outage / first run
	if len(unseen) > maxDelivery {
		unseen = unseen[:maxDelivery]
	}

	if len(unseen) == 0 {
		fmt.Println("No new items — nothing to send.")
		return
	}

	fmt.Printf("Found %d new items across %d feeds.\n", len(unseen), len(feeds))

	if os.Getenv("ENABLE_TELEGRAM") != "true" {
		for _, item := range unseen {
			fmt.Printf("[%s] %s\n  %s\n", item.Meta["feed"], item.Title, item.URL)
		}
		fmt.Println("(Telegram disabled — set ENABLE_TELEGRAM=true to send)")
		return
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	var chatID int64
	fmt.Sscanf(os.Getenv("TELEGRAM_CHAT_ID"), "%d", &chatID)
	if token == "" || chatID == 0 {
		log.Fatal("ENABLE_TELEGRAM=true but TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is missing")
	}

	formatter := &rssformatter.Formatter{}
	sender := &telegram.Sender{Token: token, ChatID: chatID}

	message := formatter.Format(unseen)
	if err := sender.Send(message); err != nil {
		log.Fatalf("send: %v", err)
	}

	// Only mark seen after successful delivery
	if err := dedup.MarkSeen(unseen); err != nil {
		log.Printf("warning: failed to mark items seen: %v", err)
	}

	fmt.Printf("[OK] Delivered %d items.\n", len(unseen))
}
