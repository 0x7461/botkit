package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/0x7461/botkit/bot"
	"github.com/0x7461/botkit/bot/curate"
	"github.com/0x7461/botkit/config"
	rssformatter "github.com/0x7461/botkit/formatters/rss"
	"github.com/0x7461/botkit/senders/telegram"
	"github.com/0x7461/botkit/sources/rss"
)

var defaultFeeds = []rss.FeedConfig{
	{Name: "HN Best", URL: "https://hnrss.org/best", MaxItems: 10, DiscussionLabel: "HN"},
	{Name: "Lobsters", URL: "https://lobste.rs/rss", MaxItems: 10},
	{Name: "Techmeme", URL: "https://techmeme.com/feed.xml", MaxItems: 10},
	{Name: "Dan Luu", URL: "https://danluu.com/atom.xml", MaxItems: 5},
	{Name: "Julia Evans", URL: "https://jvns.ca/atom.xml", MaxItems: 5},
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	cfg := config.RssBotConfig{}
	cfg.Source.MaxDelivery = 50
	if err := config.Load("rss-bot", &cfg); err != nil {
		log.Printf("Warning: could not load rss-bot config: %v", err)
	}

	feeds := defaultFeeds
	if len(cfg.Source.Feeds) > 0 {
		feeds = make([]rss.FeedConfig, len(cfg.Source.Feeds))
		for i, f := range cfg.Source.Feeds {
			feeds[i] = rss.FeedConfig{
				Name:            f.Name,
				URL:             f.URL,
				MaxItems:        f.MaxItems,
				DiscussionLabel: f.DiscussionLabel,
				SkipCurate:      f.SkipCurate,
			}
		}
	}
	maxDelivery := cfg.Source.MaxDelivery

	// Deduplication DB
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}
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
	fetched := len(items)

	// Filter seen
	unseen, err := dedup.Filter(items)
	if err != nil {
		log.Fatalf("dedup filter: %v", err)
	}
	fmt.Printf("fetched %d, deduped to %d\n", fetched, len(unseen))

	// Split: skip_curate feeds always shipped; rest go through the LLM ranker.
	var blogs, curatable []bot.Item
	for _, it := range unseen {
		if it.Meta["skip_curate"] == "true" {
			blogs = append(blogs, it)
		} else {
			curatable = append(curatable, it)
		}
	}

	// Curate (or pass-through on failure)
	picks := curatable
	if cfg.Source.Curate.Enabled && len(curatable) > 0 {
		curator := buildCurator(cfg.Source.Curate)
		if curator != nil {
			ranked, err := curator.Curate(curatable, cfg.Source.Curate.Target)
			if err != nil {
				fmt.Printf("curate: all backends failed, passing through %d items: %v\n", len(curatable), err)
			} else {
				picks = ranked
			}
		}
	}

	final := append(blogs, picks...)

	// Cap to avoid flooding after outage / first run / curation pass-through
	if len(final) > maxDelivery {
		final = final[:maxDelivery]
	}

	if len(final) == 0 {
		fmt.Println("No new items — nothing to send.")
		return
	}

	fmt.Printf("delivering: %d blogs + %d picks = %d items\n", len(blogs), len(picks), len(final))

	if os.Getenv("ENABLE_TELEGRAM") != "true" {
		for _, item := range final {
			fmt.Printf("[%s] %s\n  %s\n", item.Meta["feed"], item.Title, item.URL)
		}
		fmt.Println("(Telegram disabled — set ENABLE_TELEGRAM=true to send)")
		return
	}

	token := bot.FirstNonEmpty(os.Getenv("BOT_RSS__TOKEN"), os.Getenv("TELEGRAM_BOT_TOKEN"))
	var chatID int64
	chatStr := bot.FirstNonEmpty(os.Getenv("BOT_RSS__CHAT"), os.Getenv("TELEGRAM_CHAT_ID"))
	if chatStr != "" {
		if _, err := fmt.Sscanf(chatStr, "%d", &chatID); err != nil {
			log.Fatalf("invalid chat ID %q: %v", chatStr, err)
		}
	}
	if token == "" || chatID == 0 {
		log.Fatal("ENABLE_TELEGRAM=true but BOT_RSS__TOKEN/BOT_RSS__CHAT (or TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID) is missing")
	}

	formatter := &rssformatter.Formatter{}
	sender := &telegram.Sender{Token: token, ChatID: chatID}

	messages := formatter.FormatAll(final)
	for _, msg := range messages {
		if err := sender.Send(msg); err != nil {
			log.Fatalf("send: %v", err)
		}
	}

	// Only mark seen after successful delivery
	if err := dedup.MarkSeen(final); err != nil {
		log.Printf("warning: failed to mark items seen: %v", err)
	}

	fmt.Printf("[OK] Delivered %d items in %d messages.\n", len(final), len(messages))
}

func buildCurator(cc config.CurateConfig) curate.Curator {
	timeout := time.Duration(cc.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	primary := backendFor(cc.Backend, cc.Model, timeout)
	fallback := backendFor(cc.FallbackBackend, cc.FallbackModel, timeout)
	switch {
	case primary == nil && fallback == nil:
		fmt.Println("curate: no backend configured, skipping")
		return nil
	case fallback == nil:
		return &curate.ChainCurator{Curators: []curate.Curator{primary}}
	case primary == nil:
		return &curate.ChainCurator{Curators: []curate.Curator{fallback}}
	}
	return &curate.ChainCurator{Curators: []curate.Curator{primary, fallback}}
}

func backendFor(name, model string, timeout time.Duration) curate.Curator {
	if name == "" || model == "" {
		return nil
	}
	switch name {
	case "claude-code":
		return &curate.ClaudeCodeCurator{Model: model, Timeout: timeout}
	case "ollama":
		return &curate.OllamaCurator{Model: model, Timeout: timeout}
	default:
		fmt.Printf("curate: unknown backend %q, skipping\n", name)
		return nil
	}
}
