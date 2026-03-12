package rss

import (
	"fmt"
	"strings"

	"github.com/0x7461/botkit/bot"
)

// Formatter groups RSS items by feed and formats them as Telegram HTML.
type Formatter struct{}

// Format satisfies the bot.Formatter interface by joining all feed messages.
func (f *Formatter) Format(items []bot.Item) string {
	return strings.Join(f.FormatAll(items), "\n")
}

// FormatAll formats items as one Telegram HTML message per feed.
func (f *Formatter) FormatAll(items []bot.Item) []string {
	if len(items) == 0 {
		return nil
	}

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

	var messages []string
	for _, feed := range order {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📰 <b>%s</b>\n", escapeHTML(feed)))
		for _, item := range groups[feed] {
			line := fmt.Sprintf("• <a href=\"%s\">%s</a>", item.URL, escapeHTML(item.Title))
			if disc := item.Meta["discussion"]; disc != "" {
				label := item.Meta["discussion_label"]
				if label == "" {
					label = "Discussion"
				}
				line += fmt.Sprintf(" · <a href=\"%s\">%s</a>", disc, escapeHTML(label))
			}
			sb.WriteString(line + "\n")
		}
		messages = append(messages, sb.String())
	}
	return messages
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
