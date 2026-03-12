package markdown

import (
	"fmt"
	"strings"

	"github.com/0x7461/botkit/bot"
)

// Formatter formats items into Telegram HTML messages.
type Formatter struct {
	Title string // e.g. "GitHub Trending — Weekly Report"
}

func (f *Formatter) Format(items []bot.Item) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", escapeHTML(f.Title)))
	sb.WriteString(fmt.Sprintf("Found %d repositories:\n\n", len(items)))

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", i+1, escapeHTML(item.Title)))
		sb.WriteString(fmt.Sprintf("<a href=\"%s\">View on GitHub</a>\n\n", item.URL))

		if item.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", escapeHTML(item.Description)))
		}
		if lang := item.Meta["language"]; lang != "" {
			sb.WriteString(fmt.Sprintf("Language: <code>%s</code>\n", escapeHTML(lang)))
		}
		if stars := item.Meta["stars"]; stars != "" {
			sb.WriteString(fmt.Sprintf("Stars: %s\n", escapeHTML(stars)))
		}

		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
