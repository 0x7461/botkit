package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const contextWindow = 20

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using environment variables")
	}

	token := os.Getenv("BOT_AI__TOKEN")
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if token == "" {
		log.Fatal("BOT_AI__TOKEN is required")
	}

	// Restrict to specific chat for security
	var allowedChat int64
	if chatStr := os.Getenv("BOT_AI__CHAT"); chatStr != "" {
		fmt.Sscanf(chatStr, "%d", &allowedChat)
	}

	registry := NewRegistry()

	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".local", "share", "botkit", "ai-agent.db")

	history, err := NewHistory(dbPath)
	if err != nil {
		log.Fatalf("history init: %v", err)
	}
	defer history.Close()

	bot := &TelegramBot{Token: token}

	// Log available models
	names := modelNames(registry)
	log.Printf("AI agent started — models: %s (default: %s)", strings.Join(names, ", "), registry.DefaultModel)

	for {
		updates, err := bot.GetUpdates()
		if err != nil {
			log.Printf("poll error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			bot.Offset = update.UpdateID + 1

			msg := update.Message
			if msg == nil || msg.Text == "" {
				continue
			}

			if allowedChat != 0 && msg.Chat.ID != allowedChat {
				log.Printf("ignored message from chat %d", msg.Chat.ID)
				continue
			}

			handleMessage(bot, registry, history, msg)
		}
	}
}

func handleMessage(bot *TelegramBot, registry *ModelRegistry, history *History, msg *Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	switch {
	case text == "/start":
		names := modelNames(registry)
		greeting := "Hi! I'm your personal AI assistant.\n\n" +
			"Just send me a message and I'll reply.\n\n" +
			"Commands:\n" +
			"/clear — reset conversation\n" +
			"/model — show current model\n" +
			"/model " + strings.Join(names, "|") + " — switch model"
		bot.SendMessage(chatID, greeting)
		return

	case text == "/clear":
		bot.SendMessage(chatID, "This will delete all conversation history. Send /clear confirm to proceed.")
		return

	case text == "/clear confirm":
		history.Clear(chatID)
		bot.SendMessage(chatID, "Conversation cleared.")
		return

	case strings.HasPrefix(text, "/model"):
		parts := strings.Fields(text)
		names := modelNames(registry)
		if len(parts) < 2 {
			current := history.GetModel(chatID, registry.DefaultModel)
			bot.SendMessage(chatID, fmt.Sprintf("Current: %s\nAvailable: %s", current, strings.Join(names, ", ")))
			return
		}
		name := parts[1]
		if _, ok := registry.Models[name]; !ok {
			bot.SendMessage(chatID, fmt.Sprintf("Unknown model. Available: %s", strings.Join(names, ", ")))
			return
		}
		history.SetModel(chatID, name)
		bot.SendMessage(chatID, fmt.Sprintf("Switched to %s.", name))
		return
	}

	// Regular message — send to LLM
	model := history.GetModel(chatID, registry.DefaultModel)
	history.Add(chatID, "user", text, model)

	messages, err := history.Get(chatID, contextWindow)
	if err != nil {
		log.Printf("history error: %v", err)
		bot.SendMessage(chatID, "Error reading history.")
		return
	}

	response, err := registry.Chat(model, messages)
	if err != nil {
		log.Printf("llm error: %v", err)
		bot.SendMessage(chatID, fmt.Sprintf("Error: %v", err))
		return
	}

	history.Add(chatID, "assistant", response, model)
	bot.SendMessage(chatID, response)
}

func modelNames(registry *ModelRegistry) []string {
	var names []string
	for name := range registry.Models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
