package rss

import (
	"fmt"
	"strings"

	"github.com/0x7461/botkit/bot"
)

// Formatter groups RSS items by feed and formats them as Telegram Markdown.
type Formatter struct{}

func (f *Formatter) Format(items []bot.Item) string {
	if len(items) == 0 {
		return ""
	}

	// Group by feed, preserving order of first appearance
	order := []string{}
	groups := map[string][]bot.Item{}
	for _, item := range items {
		feed := item.Meta["feed"]
		if feed == "" {
			feed = "RSS"
		}
		if _, ok := groups[feed]; !ok {
			order = append(order, feed)
		}
		groups[feed] = append(groups[feed], item)
	}

	var sb strings.Builder
	sb.WriteString("📰 *RSS Digest*\n")

	for _, feed := range order {
		feedItems := groups[feed]
		sb.WriteString(fmt.Sprintf("\n*%s*\n", escapeMarkdown(feed)))
		for _, item := range feedItems {
			title := escapeMarkdown(item.Title)
			sb.WriteString(fmt.Sprintf("• [%s](%s)\n", title, item.URL))
		}
	}

	return sb.String()
}

// escapeMarkdown escapes Telegram MarkdownV1 special characters.
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}
