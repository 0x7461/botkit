package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/0x7461/botkit/config"
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

func fmtTZ(offset int) string {
	if offset >= 0 {
		return fmt.Sprintf("UTC+%d", offset)
	}
	return fmt.Sprintf("UTC%d", offset)
}

// Config lives at ~/.config/botkit/nagger.json (shared with cmd/nagger).
// Defaults match the migrated values; a missing file leaves them in place.
func readNaggerConfig() (weekday, hour, tzOffset int, err error) {
	cfg := config.NaggerConfig{ResetWeekday: 0, ResetHour: 11, ResetTZOffset: 7}
	if err := config.Load("nagger", &cfg); err != nil {
		return 0, 0, 0, err
	}
	return cfg.ResetWeekday, cfg.ResetHour, cfg.ResetTZOffset, nil
}

func writeNaggerConfig(weekday, hour, tzOffset int) error {
	return config.Save("nagger", config.NaggerConfig{
		ResetWeekday:  weekday,
		ResetHour:     hour,
		ResetTZOffset: tzOffset,
	})
}

func handleNagger(bot *TelegramBot, chatID int64, text string) {
	parts := strings.Fields(text)

	// /nagger — show current
	if len(parts) < 3 {
		weekday, hour, tzOffset, err := readNaggerConfig()
		if err != nil {
			bot.SendMessage(chatID, fmt.Sprintf("Error reading config: %v", err))
			return
		}
		bot.SendMessage(chatID, fmt.Sprintf(
			"Nagger reset: %s %d:00 %s\n\nUpdate: /nagger <day> <hour>\nExample: /nagger friday 19",
			dayLabels[weekday], hour, fmtTZ(tzOffset),
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

	_, _, tzOffset, err := readNaggerConfig()
	if err != nil {
		bot.SendMessage(chatID, fmt.Sprintf("Error reading config: %v", err))
		return
	}

	if err := writeNaggerConfig(weekday, hour, tzOffset); err != nil {
		bot.SendMessage(chatID, fmt.Sprintf("Error writing config: %v", err))
		return
	}

	log.Printf("nagger reset updated: %s %d:00 %s", dayLabels[weekday], hour, fmtTZ(tzOffset))
	bot.SendMessage(chatID, fmt.Sprintf("Updated nagger reset to %s %d:00 %s.", dayLabels[weekday], hour, fmtTZ(tzOffset)))
}
