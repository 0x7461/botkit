package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ClaudeCodeClient struct{}

func (c *ClaudeCodeClient) Chat(model string, messages []ChatMessage) (string, error) {
	prompt := buildPrompt(messages)

	args := []string{"-p", prompt, "--model", model, "--output-format", "text", "--allowedTools", ""}
	cmd := exec.Command("claude", args...)
	cmd.Env = filterEnv("CLAUDECODE")
	cmd.Dir = os.TempDir()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude-code: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude-code: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func buildPrompt(messages []ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}

	// Single message — no history needed
	if len(messages) == 1 {
		return messages[0].Content
	}

	// Build conversation transcript for context
	// Include model name so Claude knows which agent responded previously
	var sb strings.Builder
	sb.WriteString("Previous conversation:\n")
	for _, m := range messages[:len(messages)-1] {
		label := m.Role
		if m.Role == "assistant" && m.Model != "" {
			label = fmt.Sprintf("assistant (%s)", m.Model)
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", label, m.Content))
	}
	sb.WriteString("\nRespond to this message:\n")
	sb.WriteString(messages[len(messages)-1].Content)
	return sb.String()
}

func filterEnv(exclude string) []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, exclude+"=") {
			env = append(env, e)
		}
	}
	return env
}

func (c *ClaudeCodeClient) DefaultModel() string {
	return "claude-haiku-4-5-20251001"
}

func (c *ClaudeCodeClient) Models() map[string]string {
	return map[string]string{
		"haiku":  "claude-haiku-4-5-20251001",
		"sonnet": "claude-sonnet-4-6",
		"opus":   "claude-opus-4-6",
	}
}
