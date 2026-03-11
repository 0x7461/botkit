package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type History struct {
	db *sql.DB
}

type ChatMessage struct {
	Role    string
	Content string
	Model   string // friendly model name, stored for context
}

func NewHistory(path string) (*History, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("history: create dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("history: open db: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER NOT NULL,
		role    TEXT NOT NULL,
		content TEXT NOT NULL,
		model   TEXT NOT NULL DEFAULT '',
		ts      DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return nil, fmt.Errorf("history: create table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		chat_id INTEGER PRIMARY KEY,
		model   TEXT NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("history: create settings: %w", err)
	}

	return &History{db: db}, nil
}

func (h *History) Add(chatID int64, role, content, model string) error {
	_, err := h.db.Exec(
		`INSERT INTO messages (chat_id, role, content, model) VALUES (?, ?, ?, ?)`,
		chatID, role, content, model,
	)
	return err
}

func (h *History) Get(chatID int64, limit int) ([]ChatMessage, error) {
	rows, err := h.db.Query(
		`SELECT role, content, model FROM messages WHERE chat_id = ? ORDER BY id DESC LIMIT ?`,
		chatID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.Role, &m.Content, &m.Model); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}

	// Reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (h *History) Clear(chatID int64) error {
	_, err := h.db.Exec(`DELETE FROM messages WHERE chat_id = ?`, chatID)
	return err
}

func (h *History) GetModel(chatID int64, defaultModel string) string {
	var model string
	err := h.db.QueryRow(`SELECT model FROM settings WHERE chat_id = ?`, chatID).Scan(&model)
	if err != nil {
		return defaultModel
	}
	return model
}

func (h *History) SetModel(chatID int64, model string) error {
	_, err := h.db.Exec(
		`INSERT INTO settings (chat_id, model) VALUES (?, ?) ON CONFLICT(chat_id) DO UPDATE SET model = ?`,
		chatID, model, model,
	)
	return err
}

func (h *History) Close() error {
	return h.db.Close()
}
