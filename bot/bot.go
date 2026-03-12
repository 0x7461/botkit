package bot

import "fmt"

// Item is a generic piece of content returned by a Source.
type Item struct {
	Title       string
	URL         string
	Description string
	Meta        map[string]string // flexible key-value for extras (stars, language, etc.)
}

// Source fetches items from an external data source.
type Source interface {
	Fetch() ([]Item, error)
}

// Formatter formats a list of items into a message string.
type Formatter interface {
	Format(items []Item) string
}

// Sender delivers a formatted message to a destination.
type Sender interface {
	Send(message string) error
}

// Bot wires a Source, Formatter, and Sender together.
type Bot struct {
	Source    Source
	Formatter Formatter
	Sender    Sender
}

// FirstNonEmpty returns the first non-empty string from the arguments.
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Run fetches, formats, and sends.
func (b *Bot) Run() error {
	items, err := b.Source.Fetch()
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	message := b.Formatter.Format(items)
	if err := b.Sender.Send(message); err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}
