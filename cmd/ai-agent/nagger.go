package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var dayNames = map[string]int{
	"monday": 0, "mon": 0,
	"tuesday": 1, "tue": 1,
	"wednesday": 2, "wed": 2,
	"thursday": 3, "thu": 3,
	"friday": 4, "fri": 4,
	"saturday": 5, "sat": 5,
	"sunday": 6, "sun": 6,
}

var dayLabels = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

func naggerConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, "projects", "nagger", "config.toml"), nil
}

func mustNaggerConfigPath() string {
	p, err := naggerConfigPath()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return p
}

func readNaggerConfig() (weekday, hour int, err error) {
	data, err := os.ReadFile(mustNaggerConfigPath())
	if err != nil {
		return 0, 0, err
	}
	weekday, hour = -1, -1
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		k, v, _ := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// strip inline comment
		if i := strings.Index(v, "#"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		n, _ := strconv.Atoi(v)
		switch k {
		case "reset_weekday":
			weekday = n
		case "reset_hour":
			hour = n
		}
	}
	if weekday < 0 || hour < 0 {
		return 0, 0, fmt.Errorf("incomplete config")
	}
	return weekday, hour, nil
}

func writeNaggerConfig(weekday, hour int) error {
	content := fmt.Sprintf(
		"# Weekly reset time (when Anthropic resets the quota)\n"+
			"# Update this if the reset time drifts\n"+
			"reset_weekday = %d   # %s (Monday=0)\n"+
			"reset_hour = %d\n",
		weekday, dayLabels[weekday], hour,
	)
	return os.WriteFile(mustNaggerConfigPath(), []byte(content), 0644)
}

func handleNagger(bot *TelegramBot, chatID int64, text string) {
	parts := strings.Fields(text)

	// /nagger — show current
	if len(parts) < 3 {
		weekday, hour, err := readNaggerConfig()
		if err != nil {
			bot.SendMessage(chatID, fmt.Sprintf("Error reading config: %v", err))
			return
		}
		bot.SendMessage(chatID, fmt.Sprintf(
			"Nagger reset: %s %d:00 KST\n\nUpdate: /nagger <day> <hour>\nExample: /nagger friday 20",
			dayLabels[weekday], hour,
		))
		return
	}

	// /nagger <day> <hour>
	dayStr := strings.ToLower(parts[1])
	weekday, ok := dayNames[dayStr]
	if !ok {
		bot.SendMessage(chatID, "Unknown day. Use: monday, tuesday, ..., sunday (or mon, tue, ...)")
		return
	}

	hour, err := strconv.Atoi(parts[2])
	if err != nil || hour < 0 || hour > 23 {
		bot.SendMessage(chatID, "Hour must be 0-23.")
		return
	}

	if err := writeNaggerConfig(weekday, hour); err != nil {
		bot.SendMessage(chatID, fmt.Sprintf("Error writing config: %v", err))
		return
	}

	log.Printf("nagger reset updated: %s %d:00 KST", dayLabels[weekday], hour)
	bot.SendMessage(chatID, fmt.Sprintf("Updated nagger reset to %s %d:00 KST.", dayLabels[weekday], hour))
}
