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

// ScoutConfig holds configuration for the combined GitHub + HN scout bot.
type ScoutConfig struct {
	GitHub struct {
		Period    string `json:"period"`
		Summarize bool   `json:"summarize"`
		Limit     int    `json:"limit"`
	} `json:"github"`
	HN struct {
		Days       int    `json:"days"`
		Annotate   bool   `json:"annotate"`
		Model      string `json:"model"`
		TimeoutSec int    `json:"timeout_seconds"`
	} `json:"hn"`
	Formatter struct {
		Title string `json:"title"`
	} `json:"formatter"`
}

// SummarizeConfig controls the per-item one-line summary pass before send.
type SummarizeConfig struct {
	Enabled        bool   `json:"enabled"`
	Model          string `json:"model"`           // backend-specific model id (default sonnet)
	TimeoutSeconds int    `json:"timeout_seconds"` // wall-clock cap for the batched call
}

// RssBotConfig holds configuration for the RSS digest bot.
type RssBotConfig struct {
	Source struct {
		MaxDelivery int             `json:"max_delivery"`
		Feeds       []FeedEntry     `json:"feeds"`
		Curate      CurateConfig    `json:"curate"`
		Summarize   SummarizeConfig `json:"summarize"`
	} `json:"source"`
}

// NaggerConfig holds the weekly-quota-reset anchor for the nagger bot.
// Used only by the fallback cycle-day calc; the live path prefers the API's
// resets_at from ~/.local/share/nagger/rate-limits.json.
type NaggerConfig struct {
	ResetWeekday  int `json:"reset_weekday"`  // Monday=0
	ResetHour     int `json:"reset_hour"`     // 0-23
	ResetTZOffset int `json:"reset_tz_offset"` // hours from UTC
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

// Save writes v as JSON to ~/.config/botkit/<name>.json, creating the dir.
func Save(name string, v any) error {
	d := dir()
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, name+".json"), append(data, '\n'), 0o644)
}

func dir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "botkit")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "botkit")
}
