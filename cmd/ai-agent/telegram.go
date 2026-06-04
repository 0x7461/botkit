package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const telegramAPI = "https://api.telegram.org/bot"
const maxMessageLen = 4096

// Long-poll timeout is 30s, so HTTP timeout must be longer to avoid
// cutting off the connection before Telegram responds.
var telegramHTTPClient = &http.Client{Timeout: 60 * time.Second}

type TelegramBot struct {
	Token  string
	Offset int64
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

// scrub removes the bot token from an error string. Go's net/http wraps the
// request URL (which carries the token in its path) into *url.Error, so a bare
// network error would otherwise log the token. See secret-audit #47.
func (b *TelegramBot) scrub(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(strings.ReplaceAll(err.Error(), b.Token, "<redacted>"))
}

func (b *TelegramBot) GetUpdates(ctx context.Context) ([]Update, error) {
	url := fmt.Sprintf("%s%s/getUpdates?timeout=30&offset=%d", telegramAPI, b.Token, b.Offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, b.scrub(err)
	}
	resp, err := telegramHTTPClient.Do(req)
	if err != nil {
		return nil, b.scrub(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Ok     bool     `json:"ok"`
		Result []Update `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if !result.Ok {
		return nil, fmt.Errorf("telegram getUpdates failed")
	}
	return result.Result, nil
}

func (b *TelegramBot) SendMessage(chatID int64, text string) error {
	for _, chunk := range splitText(text) {
		if err := b.sendChunk(chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (b *TelegramBot) sendChunk(chatID int64, text string) error {
	url := fmt.Sprintf("%s%s/sendMessage", telegramAPI, b.Token)
	payload, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	resp, err := telegramHTTPClient.Post(url, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return b.scrub(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}
	json.Unmarshal(body, &result)
	if !result.Ok {
		return fmt.Errorf("telegram: %s", result.Description)
	}
	return nil
}

func splitText(text string) []string {
	if len(text) <= maxMessageLen {
		return []string{text}
	}
	lines := strings.Split(text, "\n")
	var chunks []string
	current := ""
	for _, line := range lines {
		// Handle single lines longer than the limit by hard-splitting them
		for len(line) > maxMessageLen {
			if current != "" {
				chunks = append(chunks, current)
				current = ""
			}
			chunks = append(chunks, line[:maxMessageLen])
			line = line[maxMessageLen:]
		}
		if len(current)+len(line)+1 > maxMessageLen && current != "" {
			chunks = append(chunks, current)
			current = ""
		}
		if current != "" {
			current += "\n"
		}
		current += line
	}
	if current != "" {
		chunks = append(chunks, current)
	}
	return chunks
}
