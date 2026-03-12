package rss

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/0x7461/botkit/bot"
)

// Deduplicator tracks seen items via SQLite to avoid re-delivering them.
type Deduplicator struct {
	db *sql.DB
}

// NewDeduplicator opens (or creates) the seen-items DB at the given path.
func NewDeduplicator(path string) (*Deduplicator, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("dedup: create dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("dedup: open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS seen (
		guid    TEXT PRIMARY KEY,
		feed    TEXT NOT NULL,
		seen_at INTEGER NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("dedup: create table: %w", err)
	}

	// Prune entries older than 90 days to keep the table bounded
	cutoff := time.Now().Add(-90 * 24 * time.Hour).Unix()
	if _, err := db.Exec(`DELETE FROM seen WHERE seen_at < ?`, cutoff); err != nil {
		return nil, fmt.Errorf("dedup: prune old entries: %w", err)
	}

	return &Deduplicator{db: db}, nil
}

// Filter returns only items whose GUID has not been seen before.
func (d *Deduplicator) Filter(items []bot.Item) ([]bot.Item, error) {
	var unseen []bot.Item
	for _, item := range items {
		guid := item.Meta["guid"]
		if guid == "" {
			guid = item.URL
		}
		var count int
		err := d.db.QueryRow(`SELECT COUNT(*) FROM seen WHERE guid = ?`, guid).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("dedup: query %q: %w", guid, err)
		}
		if count == 0 {
			unseen = append(unseen, item)
		}
	}
	return unseen, nil
}

// MarkSeen records items as delivered. Call after successful Send.
func (d *Deduplicator) MarkSeen(items []bot.Item) error {
	now := time.Now().Unix()
	for _, item := range items {
		guid := item.Meta["guid"]
		if guid == "" {
			guid = item.URL
		}
		feed := item.Meta["feed"]
		_, err := d.db.Exec(
			`INSERT OR IGNORE INTO seen (guid, feed, seen_at) VALUES (?, ?, ?)`,
			guid, feed, now,
		)
		if err != nil {
			return fmt.Errorf("dedup: mark %q: %w", guid, err)
		}
	}
	return nil
}

// Close releases the database connection.
func (d *Deduplicator) Close() error {
	return d.db.Close()
}
