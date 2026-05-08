// Package config loads per-bot JSON config from ~/.config/botkit/<name>.json.
// If the file doesn't exist, callers keep their hardwired defaults.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FeedEntry mirrors rss.FeedConfig for JSON serialization.
type FeedEntry struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	MaxItems        int    `json:"max_items"`
	DiscussionLabel string `json:"discussion_label,omitempty"`
}

// GhBotConfig holds configuration for the GitHub trending bot.
type GhBotConfig struct {
	Source struct {
		Period    string `json:"period"`
		Summarize bool   `json:"summarize"`
	} `json:"source"`
	Formatter struct {
		Title string `json:"title"`
	} `json:"formatter"`
}

// RssBotConfig holds configuration for the RSS digest bot.
type RssBotConfig struct {
	Source struct {
		MaxDelivery int         `json:"max_delivery"`
		Feeds       []FeedEntry `json:"feeds"`
	} `json:"source"`
}

// Load reads ~/.config/botkit/<name>.json into v.
// If the file does not exist, v is unchanged and nil is returned.
func Load(name string, v any) error {
	path := filepath.Join(dir(), name+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func dir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "botkit")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "botkit")
}
