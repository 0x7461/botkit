package main

import (
	"os"
	"os/exec"
)

// LLM is the interface for language model backends.
type LLM interface {
	Chat(model string, messages []ChatMessage) (string, error)
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

	// Claude Code — available if claude binary is in PATH
	if _, err := exec.LookPath("claude"); err == nil {
		cc := &ClaudeCodeClient{}
		for name, id := range cc.Models() {
			r.Models[name] = ModelEntry{Backend: cc, ModelID: id}
		}
		r.DefaultModel = "haiku"
	}

	// Claude API — available if ANTHROPIC_API_KEY is set
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		api := &ClaudeClient{APIKey: key}
		for name, id := range api.Models() {
			r.Models[name] = ModelEntry{Backend: api, ModelID: id}
		}
		r.DefaultModel = "haiku"
	}

	return r
}

func (r *ModelRegistry) Chat(name string, messages []ChatMessage) (string, error) {
	entry, ok := r.Models[name]
	if !ok {
		entry = r.Models[r.DefaultModel]
	}
	return entry.Backend.Chat(entry.ModelID, messages)
}
