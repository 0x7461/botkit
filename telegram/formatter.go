package telegram

import (
	"fmt"
	"strings"

	"github.com/0x7461/github-trending/bot"
)

// Formatter formats items into Telegram Markdown messages.
type Formatter struct {
	Title string // e.g. "GitHub Trending — Weekly Report"
}

func (f *Formatter) Format(items []bot.Item) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*%s*\n\n", escapeMarkdown(f.Title)))
	sb.WriteString(fmt.Sprintf("Found %d repositories:\n\n", len(items)))

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, escapeMarkdown(item.Title)))
		sb.WriteString(fmt.Sprintf("[View on GitHub](%s)\n\n", item.URL))

		if item.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", escapeMarkdown(item.Description)))
		}
		if lang := item.Meta["language"]; lang != "" {
			sb.WriteString(fmt.Sprintf("🔹 Language: `%s`\n", lang))
		}
		if stars := item.Meta["stars"]; stars != "" {
			sb.WriteString(fmt.Sprintf("⭐ %s\n", stars))
		}

		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")
	}

	return sb.String()
}

func escapeMarkdown(text string) string {
	return strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	).Replace(text)
}
