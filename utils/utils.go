package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Repo represents a trending GitHub repository
type Repo struct {
	Name        string
	Description string
	URL         string
	Stars       string
	Language    string
}

// GetTelegramChatID fetches your chat ID from Telegram bot
// You must have sent /start to the bot first!
func GetTelegramChatID(botToken string) (int64, error) {
	// Telegram API endpoint
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)

	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch updates: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}


	// Parse JSON response
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

	// Check if we got any messages
	if !result.Ok || len(result.Result) == 0 {
		return 0, fmt.Errorf("no messages found - did you send /start to the bot?")
	}

	chatID := result.Result[0].Message.Chat.ID
	firstName := result.Result[0].Message.Chat.FirstName

	fmt.Printf("[OK] Found your chat!\n")
	fmt.Printf("     Name: %s\n", firstName)
	fmt.Printf("     Chat ID: %d\n", chatID)

	return chatID, nil
}

// SendTelegramMessage sends a message to Telegram using the bot
func SendTelegramMessage(botToken string, chatID int64, message string) error {
	// Telegram API endpoint for sending messages
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	// Create form data
	data := fmt.Sprintf("chat_id=%d&text=%s&parse_mode=Markdown", chatID, message)

	// Make HTTP POST request
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	// Check response
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

// FormatAsMarkdown formats repos as standard markdown (for email)
func FormatAsMarkdown(repos []Repo) string {
	var sb strings.Builder

	sb.WriteString("# GitHub Trending - Weekly Report\n\n")
	sb.WriteString(fmt.Sprintf("Found %d trending repositories this week:\n\n", len(repos)))
	sb.WriteString("---\n\n")

	for i, repo := range repos {
		sb.WriteString(fmt.Sprintf("## %d. [%s](%s)\n\n", i+1, repo.Name, repo.URL))

		if repo.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", repo.Description))
		}

		if repo.Language != "" || repo.Stars != "" {
			sb.WriteString("**Details:**\n")
			if repo.Language != "" {
				sb.WriteString(fmt.Sprintf("- Language: %s\n", repo.Language))
			}
			if repo.Stars != "" {
				sb.WriteString(fmt.Sprintf("- Stars: %s\n", repo.Stars))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// FormatForTelegram formats repos for Telegram's markdown syntax
func FormatForTelegram(repos []Repo) string {
	var sb strings.Builder

	sb.WriteString("*GitHub Trending - Weekly Report*\n\n")
	sb.WriteString(fmt.Sprintf("Found %d trending repositories this week:\n\n", len(repos)))

	for i, repo := range repos {
		// Telegram markdown: *bold* _italic_ [link](url) `code`
		sb.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, escapeMarkdown(repo.Name)))
		sb.WriteString(fmt.Sprintf("[View on GitHub](%s)\n\n", repo.URL))

		if repo.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", escapeMarkdown(repo.Description)))
		}

		if repo.Language != "" {
			sb.WriteString(fmt.Sprintf("🔹 Language: `%s`\n", repo.Language))
		}
		if repo.Stars != "" {
			sb.WriteString(fmt.Sprintf("⭐ %s\n", repo.Stars))
		}

		sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n\n")
	}

	return sb.String()
}

// escapeMarkdown escapes special characters for Telegram markdown (legacy mode)
func escapeMarkdown(text string) string {
	// For Telegram's legacy Markdown mode, only escape: _ * [ `
	// These are the actual markdown syntax characters
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	)
	return replacer.Replace(text)
}

