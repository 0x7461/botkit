package rss

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0x7461/guild/bot"
)

// pendingRetention bounds the queue if delivery stops firing (service down,
// ollama unreachable). Two weeks is four missed deliveries — long enough that
// a normal outage loses nothing, short enough that the queue can't grow without
// limit and hand the curator a list it can't rank.
const pendingRetention = 14 * 24 * time.Hour

// initPending creates the queue of curated-but-undelivered items.
//
// It stores the whole item, not just a guid, because that is the entire point:
// a feed only exposes its newest `max_items`, so an item that is fetched today
// and delivered on Friday is gone from the feed by then. `seen` can say "we
// judged this already"; only this table can still produce it.
func (d *Deduplicator) initPending() error {
	_, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS pending (
		guid        TEXT PRIMARY KEY,
		feed        TEXT NOT NULL,
		title       TEXT NOT NULL,
		url         TEXT NOT NULL,
		description TEXT NOT NULL,
		meta        TEXT NOT NULL,
		kept_at     INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("pending: create table: %w", err)
	}
	cutoff := time.Now().Add(-pendingRetention).Unix()
	if _, err := d.db.Exec(`DELETE FROM pending WHERE kept_at < ?`, cutoff); err != nil {
		return fmt.Errorf("pending: prune: %w", err)
	}
	return nil
}

func itemGUID(item bot.Item) string {
	if g := item.Meta["guid"]; g != "" {
		return g
	}
	return item.URL
}

// Keep queues curated items for a later delivery run. Idempotent per guid, so
// re-running a curate pass on the same day adds nothing.
func (d *Deduplicator) Keep(items []bot.Item) error {
	now := time.Now().Unix()
	for _, item := range items {
		meta, err := json.Marshal(item.Meta)
		if err != nil {
			return fmt.Errorf("pending: encode meta for %q: %w", item.URL, err)
		}
		_, err = d.db.Exec(
			`INSERT OR IGNORE INTO pending
			 (guid, feed, title, url, description, meta, kept_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			itemGUID(item), item.Meta["feed"], item.Title, item.URL, item.Description, string(meta), now,
		)
		if err != nil {
			return fmt.Errorf("pending: keep %q: %w", item.URL, err)
		}
	}
	return nil
}

// Pending returns the queued items, oldest first, so a delivery reads in the
// order the days actually happened.
func (d *Deduplicator) Pending() ([]bot.Item, error) {
	rows, err := d.db.Query(
		`SELECT title, url, description, meta FROM pending ORDER BY kept_at ASC, rowid ASC`)
	if err != nil {
		return nil, fmt.Errorf("pending: query: %w", err)
	}
	defer rows.Close()

	var items []bot.Item
	for rows.Next() {
		var title, url, desc, metaJSON string
		if err := rows.Scan(&title, &url, &desc, &metaJSON); err != nil {
			return nil, fmt.Errorf("pending: scan: %w", err)
		}
		meta := map[string]string{}
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return nil, fmt.Errorf("pending: decode meta for %q: %w", url, err)
		}
		items = append(items, bot.Item{Title: title, URL: url, Description: desc, Meta: meta})
	}
	return items, rows.Err()
}

// ClearPending removes delivered items from the queue. Call only after a
// successful send: anything left queued is re-delivered next run, which is the
// behaviour a failed send needs.
func (d *Deduplicator) ClearPending(items []bot.Item) error {
	for _, item := range items {
		if _, err := d.db.Exec(`DELETE FROM pending WHERE guid = ?`, itemGUID(item)); err != nil {
			return fmt.Errorf("pending: clear %q: %w", item.URL, err)
		}
	}
	return nil
}

// PendingCount reports the queue depth, for logging a curate run that sends nothing.
func (d *Deduplicator) PendingCount() (int, error) {
	var n int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM pending`).Scan(&n)
	return n, err
}
