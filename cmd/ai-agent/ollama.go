package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ollamaHTTPClient = &http.Client{Timeout: 5 * time.Minute}

type OllamaClient struct {
	BaseURL string
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Think    bool            `json:"think"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

func (o *OllamaClient) Chat(model string, messages []ChatMessage, chatID int64) (string, error) {
	var msgs []ollamaMessage
	msgs = append(msgs, ollamaMessage{
		Role:    "system",
		Content: "You are a helpful personal assistant accessible via Telegram. Be concise — most messages are read on a phone screen. Note: previous assistant replies may have come from different models — the model name is shown in brackets when available.",
	})
	for _, m := range messages {
		content := m.Content
		if m.Role == "assistant" && m.Model != "" {
			content = fmt.Sprintf("[%s] %s", m.Model, content)
		}
		msgs = append(msgs, ollamaMessage{Role: m.Role, Content: content})
	}

	body, _ := json.Marshal(ollamaRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
		Think:    false,
	})

	url := o.BaseURL + "/api/chat"
	resp, err := ollamaHTTPClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ollama: invalid response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("ollama: %s", result.Error)
	}
	if result.Message.Content == "" {
		return "", fmt.Errorf("ollama: empty response")
	}
	return result.Message.Content, nil
}

func (o *OllamaClient) DefaultModel() string {
	return "qwen3.5:4b"
}

func (o *OllamaClient) Models() map[string]string {
	return map[string]string{
		"qwen3.5":  "qwen3.5:4b",
		"coder":    "qwen2.5-coder:3b",
		"deepseek": "deepseek-r1:1.5b",
	}
}
