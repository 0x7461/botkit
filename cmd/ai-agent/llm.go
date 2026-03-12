package main

import (
	"log"
	"os"
	"os/exec"
)

// LLM is the interface for language model backends.
type LLM interface {
	Chat(model string, messages []ChatMessage, chatID int64) (string, error)
}

// ModelEntry maps a friendly name to a backend + model ID.
type ModelEntry struct {
	Backend LLM
	ModelID string
}

// ModelRegistry holds all available models across backends.
type ModelRegistry struct {
	Models       map[string]ModelEntry
	DefaultModel string // friendly name
}

// NewRegistry builds a registry from all available backends.
func NewRegistry() *ModelRegistry {
	r := &ModelRegistry{
		Models:       make(map[string]ModelEntry),
		DefaultModel: "qwen3.5",
	}

	// Ollama — always available
	ollama := &OllamaClient{BaseURL: "http://localhost:11434"}
	if url := os.Getenv("OLLAMA_URL"); url != "" {
		ollama.BaseURL = url
	}
	for name, id := range ollama.Models() {
		r.Models[name] = ModelEntry{Backend: ollama, ModelID: id}
	}

	// Claude Code — preferred when claude binary is in PATH (supports /tools)
	// Claude API — fallback when only ANTHROPIC_API_KEY is set
	// Only one Claude backend is registered to avoid silent overwrites.
	if _, err := exec.LookPath("claude"); err == nil {
		cc := &ClaudeCodeClient{ToolsEnabled: make(map[int64]bool)}
		for name, id := range cc.Models() {
			r.Models[name] = ModelEntry{Backend: cc, ModelID: id}
		}
		r.DefaultModel = "haiku"
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			log.Println("note: ANTHROPIC_API_KEY set but claude binary found — using claude-code backend (supports /tools)")
		}
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		api := &ClaudeClient{APIKey: key}
		for name, id := range api.Models() {
			r.Models[name] = ModelEntry{Backend: api, ModelID: id}
		}
		r.DefaultModel = "haiku"
	}

	return r
}

func (r *ModelRegistry) Chat(name string, messages []ChatMessage, chatID int64) (string, error) {
	entry, ok := r.Models[name]
	if !ok {
		entry = r.Models[r.DefaultModel]
	}
	return entry.Backend.Chat(entry.ModelID, messages, chatID)
}

