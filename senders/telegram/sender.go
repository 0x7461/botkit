package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Sender delivers messages via the Telegram Bot API.
type Sender struct {
	Token  string
	ChatID int64
}

const telegramMaxLen = 4096
const repoSeparator = "\n━━━━━━━━━━━━━━━━━━━━\n\n"

func (s *Sender) Send(message string) error {
	for _, chunk := range splitMessage(message) {
		if err := s.sendChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

func splitMessage(message string) []string {
	if len(message) <= telegramMaxLen {
		return []string{message}
	}

	parts := strings.Split(message, repoSeparator)
	var chunks []string
	current := ""

	for i, part := range parts {
		segment := part
		if i < len(parts)-1 {
			segment += repoSeparator
		}
		if len(current)+len(segment) > telegramMaxLen {
			if current != "" {
				chunks = append(chunks, current)
			}
			current = segment
		} else {
			current += segment
		}
	}
	if current != "" {
		chunks = append(chunks, current)
	}
	return chunks
}

func (s *Sender) sendChunk(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.Token)
	payload, err := json.Marshal(map[string]any{
		"chat_id":    s.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if !result.Ok {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}
	return nil
}

// GetChatID fetches the chat ID from the bot's recent updates.
// Requires the user to have sent /start to the bot first.
func GetChatID(token string) (int64, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", token)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch updates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			Message struct {
				Chat struct {
					ID        int64  `json:"id"`
					FirstName string `json:"first_name"`
				} `json:"chat"`
			} `json:"message"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if !result.Ok || len(result.Result) == 0 {
		return 0, fmt.Errorf("no messages found — did you send /start to the bot?")
	}

	chat := result.Result[0].Message.Chat
	fmt.Printf("[OK] Found chat: %s (ID: %d)\n", chat.FirstName, chat.ID)
	return chat.ID, nil
}
