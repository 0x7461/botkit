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
	SkipCurate      bool   `json:"skip_curate,omitempty"`
}

// CurateConfig controls the LLM ranking pass between dedup and send.
type CurateConfig struct {
	Enabled         bool   `json:"enabled"`
	Target          int    `json:"target"`            // how many items to keep after ranking
	Backend         string `json:"backend"`           // "claude-code" or "ollama"
	Model           string `json:"model"`             // backend-specific model id
	FallbackBackend string `json:"fallback_backend"`  // optional second backend
	FallbackModel   string `json:"fallback_model"`
	TimeoutSeconds  int    `json:"timeout_seconds"`   // per-backend wall-clock cap
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
		MaxDelivery int          `json:"max_delivery"`
		Feeds       []FeedEntry  `json:"feeds"`
		Curate      CurateConfig `json:"curate"`
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
