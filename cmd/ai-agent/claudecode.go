package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ClaudeCodeClient struct {
	ToolsEnabled map[int64]bool // per-chat tools state
}

const toolsAllowed = "Read,Write,Edit,Glob,Grep"

func (c *ClaudeCodeClient) Chat(model string, messages []ChatMessage, chatID int64) (string, error) {
	prompt := buildPrompt(messages)

	allowed := ""
	if c.ToolsEnabled[chatID] {
		allowed = toolsAllowed
	}
	args := []string{"-p", prompt, "--model", model, "--output-format", "text", "--allowedTools", allowed}
	cmd := exec.Command("claude", args...)
	cmd.Env = filterEnv("CLAUDECODE")
	dir := os.TempDir()
	if c.ToolsEnabled[chatID] {
		// Run from home so file paths make sense
		dir, _ = os.UserHomeDir()
	}
	cmd.Dir = dir
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
