package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const claudeAPI = "https://api.anthropic.com/v1/messages"
const claudeVersion = "2023-06-01"

type ClaudeClient struct {
	APIKey string
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *ClaudeClient) Chat(model string, messages []ChatMessage, chatID int64) (string, error) {
	var msgs []claudeMessage
	for _, m := range messages {
		msgs = append(msgs, claudeMessage{Role: m.Role, Content: m.Content})
	}

	body, _ := json.Marshal(claudeRequest{
		Model:     model,
		MaxTokens: 4096,
		System:    "You are a helpful personal assistant accessible via Telegram. Be concise — most messages are read on a phone screen.",
		Messages:  msgs,
	})

	req, _ := http.NewRequest("POST", claudeAPI, bytes.NewReader(body))
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", claudeVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result claudeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("claude: invalid response: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("claude: %s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("claude: empty response")
	}

	var text string
	for _, block := range result.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text, nil
}

func (c *ClaudeClient) DefaultModel() string {
	return "claude-haiku-4-5-20251001"
}

func (c *ClaudeClient) Models() map[string]string {
	return map[string]string{
		"haiku":  "claude-haiku-4-5-20251001",
		"sonnet": "claude-sonnet-4-6",
		"opus":   "claude-opus-4-6",
	}
}
