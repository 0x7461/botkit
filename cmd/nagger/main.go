// nagger — one-shot Telegram nudge to stay on pace with the weekly Claude quota.
// Ported from the standalone Python project (~/projects/nagger) into botkit.
// Run hourly 08–22 by runit + snooze; dedup'd to one message per day.
//
// Inputs:
//   - ~/.local/share/nagger/rate-limits.json  (written by ~/.claude/statusline.sh
//     after every CC response — external producer, do not move)
//   - ~/.config/botkit/nagger.json            (schedule anchor; fallback cycle calc)
//   - ~/.local/share/nagger/last-sent         (daily dedup state, owned here)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0x7461/botkit/bot"
	"github.com/0x7461/botkit/config"
	"github.com/0x7461/botkit/senders/telegram"
)

// Pace cycle: index = day-1 → (label, target % weekly usage by end of that day).
var cycle = []struct {
	Label  string
	Target int
}{
	{"Heavy Build", 18},
	{"Heavy Build", 34},
	{"Heavy Build", 50},
	{"Refinement", 63},
	{"Refinement", 75},
	{"Buffer", 85},
	{"Careful Sprint", 95},
}

func stateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nagger")
}

type rateLimits struct {
	SevenDay         int   `json:"seven_day"`
	SevenDayResetsAt int64 `json:"seven_day_resets_at"`
}

func readRateLimits() (*rateLimits, bool) {
	data, err := os.ReadFile(filepath.Join(stateDir(), "rate-limits.json"))
	if err != nil {
		return nil, false
	}
	var rl rateLimits
	if err := json.Unmarshal(data, &rl); err != nil {
		return nil, false
	}
	return &rl, true
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// getCycleDay prefers the API's actual reset timestamp (rolling 7-day window);
// falls back to the configured weekday/hour anchor when the cache is missing.
func getCycleDay(resetsAt int64, cfg config.NaggerConfig) int {
	if resetsAt > 0 {
		secsRemaining := float64(resetsAt) - float64(time.Now().Unix())
		daysElapsed := 7 - secsRemaining/86400
		return clamp(int(daysElapsed)+1, 1, 7)
	}
	loc := time.FixedZone("local", cfg.ResetTZOffset*3600)
	now := time.Now().In(loc)
	weekday := (int(now.Weekday()) + 6) % 7 // Go: Sunday=0 → Monday=0
	daysSince := (weekday - cfg.ResetWeekday + 7) % 7
	lastReset := time.Date(now.Year(), now.Month(), now.Day()-daysSince, cfg.ResetHour, 0, 0, 0, loc)
	if now.Before(lastReset) {
		lastReset = lastReset.AddDate(0, 0, -7)
	}
	days := int(now.UTC().Sub(lastReset.UTC()).Hours() / 24)
	return clamp(days+1, 1, 7)
}

func formatReset(ts int64) string {
	if ts == 0 {
		return ""
	}
	delta := time.Until(time.Unix(ts, 0))
	if delta <= 0 {
		return "resets now"
	}
	days := int(delta.Hours()) / 24
	hours := int(delta.Hours()) % 24
	mins := int(delta.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("resets in %dd %dh", days, hours)
	}
	return fmt.Sprintf("resets in %dh %dm", hours, mins)
}

func todayInTZ(tzOffset int) string {
	return time.Now().In(time.FixedZone("local", tzOffset*3600)).Format("2006-01-02")
}

// --- Reminders: recurring manual-task nudges, independent of the quota dedup ---

func reminderStatePath() string {
	return filepath.Join(stateDir(), "reminders-state.json")
}

// readReminderState returns the id→last-fired-date (YYYY-MM-DD) map; empty on any error.
func readReminderState() map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(reminderStatePath())
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func writeReminderState(m map[string]string) {
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		log.Printf("warning: mkdir state dir: %v", err)
		return
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		log.Printf("warning: marshal reminder state: %v", err)
		return
	}
	if err := os.WriteFile(reminderStatePath(), append(data, '\n'), 0o644); err != nil {
		log.Printf("warning: write reminder state: %v", err)
	}
}

// remindersDue returns the reminders whose next-due date has arrived, using the
// last-fired state (falling back to Anchor when a reminder has never fired).
func remindersDue(cfg config.NaggerConfig, state map[string]string, now time.Time) []config.Reminder {
	parseDay := func(s string) (time.Time, bool) {
		t, err := time.Parse("2006-01-02", s)
		return t, err == nil
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var due []config.Reminder
	for _, r := range cfg.Reminders {
		if r.ID == "" || r.EveryDays <= 0 {
			continue
		}
		var next time.Time
		if lf, ok := state[r.ID]; ok {
			if t, ok2 := parseDay(lf); ok2 {
				next = t.AddDate(0, 0, r.EveryDays)
			}
		}
		if next.IsZero() {
			a, ok := parseDay(r.Anchor)
			if !ok {
				continue // no usable anchor and never fired — skip rather than fire blindly
			}
			next = a
		}
		if !today.Before(next) {
			due = append(due, r)
		}
	}
	return due
}

func buildSender() *telegram.Sender {
	token := bot.FirstNonEmpty(os.Getenv("BOT_NAGGER__TOKEN"), os.Getenv("TELEGRAM_BOT_TOKEN"))
	var chatID int64
	chatStr := bot.FirstNonEmpty(os.Getenv("BOT_NAGGER__CHAT"), os.Getenv("TELEGRAM_CHAT_ID"))
	if chatStr != "" {
		if _, err := fmt.Sscanf(chatStr, "%d", &chatID); err != nil {
			log.Fatalf("invalid chat ID %q: %v", chatStr, err)
		}
	}
	if token == "" || chatID == 0 {
		log.Fatal("ENABLE_TELEGRAM=true but BOT_NAGGER__TOKEN/BOT_NAGGER__CHAT (or TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID) is missing")
	}
	return &telegram.Sender{Token: token, ChatID: chatID}
}

// runReminders fires any due manual-task nudges as a single Telegram message,
// independent of the quota-nudge daily dedup (each reminder has its own cadence
// state). No-op when nothing is due. In dry-run it logs and writes no state.
func runReminders(cfg config.NaggerConfig, enabled bool, today string, now time.Time, sender *telegram.Sender) {
	state := readReminderState()
	due := remindersDue(cfg, state, now)
	if len(due) == 0 {
		return
	}
	lines := []string{"🔔 Quarterly archive reminder — not urgent, do when convenient:"}
	for _, r := range due {
		lines = append(lines, "• "+r.Message)
	}
	msg := strings.Join(lines, "\n")
	if !enabled {
		fmt.Println("[dry-run] would send reminder:")
		fmt.Println(msg)
		return
	}
	if err := sender.Send(msg); err != nil {
		log.Printf("warning: reminder send failed: %v", err)
		return // don't advance state on a failed send — retry next run
	}
	for _, r := range due {
		state[r.ID] = today
	}
	writeReminderState(state)
	fmt.Printf("Sent reminder: %d due\n", len(due))
}

func main() {
	bot.LoadEnv("nagger")

	cfg := config.NaggerConfig{ResetWeekday: 0, ResetHour: 11, ResetTZOffset: 7} // migrated defaults
	if err := config.Load("nagger", &cfg); err != nil {
		log.Printf("warning: could not load nagger config: %v", err)
	}

	enabled := os.Getenv("ENABLE_TELEGRAM") == "true"
	today := todayInTZ(cfg.ResetTZOffset)
	nowLocal := time.Now().In(time.FixedZone("local", cfg.ResetTZOffset*3600))
	lastSentPath := filepath.Join(stateDir(), "last-sent")

	// One sender for both the reminder and quota messages (only needed for real sends).
	var sender *telegram.Sender
	if enabled {
		sender = buildSender()
	}

	// Reminders run before the quota daily-dedup — they have their own per-item
	// cadence state, so a due reminder fires regardless of whether today's quota
	// nudge already went out.
	runReminders(cfg, enabled, today, nowLocal, sender)

	if enabled {
		if b, err := os.ReadFile(lastSentPath); err == nil && strings.TrimSpace(string(b)) == today {
			fmt.Println("Already sent today — skipping.")
			return
		}
	}

	rl, haveRL := readRateLimits()
	var resetsAt int64
	var actual *int
	if haveRL {
		resetsAt = rl.SevenDayResetsAt
		a := rl.SevenDay
		actual = &a
	}

	day := getCycleDay(resetsAt, cfg)
	c := cycle[day-1]
	floor := int(float64(c.Target) * 0.7)

	var paceLine string
	switch {
	case actual == nil:
		paceLine = "Actual: unknown (open CC to refresh)"
	case *actual < floor:
		paceLine = fmt.Sprintf("Actual: %d%% — under-using 💸 (target %d%%, floor %d%%)", *actual, c.Target, floor)
	case *actual <= c.Target:
		paceLine = fmt.Sprintf("Actual: %d%% — on pace ✓", *actual)
	default:
		paceLine = fmt.Sprintf("Actual: %d%% — over pace ⚠️ (target %d%%)", *actual, c.Target)
	}

	lines := []string{
		fmt.Sprintf("📊 Claude quota check — Day %d/7 (%s)", day, c.Label),
		fmt.Sprintf("Target: ~%d%% weekly usage by end of today.", c.Target),
		paceLine,
	}
	if rs := formatReset(resetsAt); rs != "" {
		lines = append(lines, strings.ToUpper(rs[:1])+rs[1:]+".")
	}
	msg := strings.Join(lines, "\n")

	if !enabled {
		fmt.Println("[dry-run] ENABLE_TELEGRAM != true — would send:")
		fmt.Println(msg)
		return
	}

	if err := sender.Send(msg); err != nil {
		log.Fatalf("send failed: %v", err)
	}
	if err := os.MkdirAll(stateDir(), 0o755); err != nil {
		log.Printf("warning: mkdir state dir: %v", err)
	}
	if err := os.WriteFile(lastSentPath, []byte(today), 0o644); err != nil {
		log.Printf("warning: write last-sent: %v", err)
	}
	fmt.Printf("Sent: Day %d/7 — %s — target %d%%\n", day, c.Label, c.Target)
}
