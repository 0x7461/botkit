package curate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/0x7461/botkit/bot"
)

// Annotation is the per-item enrichment returned by the model.
type Annotation struct {
	Summary   string `json:"summary"`
	Sentiment string `json:"sentiment"` // civil | mixed | heated | toxic
}

const annotatePrompt = `For each discussion-board post below, return:
- "summary": one neutral sentence on what the post and its discussion are about.
- "sentiment": the overall tone of the comment thread, exactly one of:
  civil, mixed, heated, toxic.

Base sentiment on the comments provided. "toxic" = personal attacks, hostility,
or pervasive bad faith; "heated" = sharp disagreement but on substance;
"mixed" = a blend; "civil" = constructive and respectful.

Return a JSON array only — no prose, no markdown fences — one object per post in
the same order:
[{"summary": "...", "sentiment": "..."}]

Posts:
%s`

// Annotate best-effort fills Meta["summary"] and Meta["sentiment"] for each
// item from its Title and Meta["context"] (e.g. top comments). On any failure
// it logs and leaves items unchanged, so callers can always ship the digest.
func Annotate(items []bot.Item, model string, timeout time.Duration) {
	if len(items) == 0 {
		return
	}
	if full, ok := claudeModelMap[model]; ok {
		model = full
	}

	var sb strings.Builder
	for i, it := range items {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i, it.Title))
		if ctx := strings.TrimSpace(it.Meta["context"]); ctx != "" {
			sb.WriteString("comments:\n" + ctx + "\n")
		}
		sb.WriteString("\n")
	}
	prompt := fmt.Sprintf(annotatePrompt, sb.String())

	out, err := runClaudeText(model, timeout, prompt)
	if err != nil {
		fmt.Printf("annotate: %v\n", err)
		return
	}
	anns, err := parseAnnotations(out)
	if err != nil {
		fmt.Printf("annotate: parse: %v (raw: %q)\n", err, truncate(out, 200))
		return
	}
	for i := range items {
		if i >= len(anns) {
			break
		}
		if anns[i].Summary != "" {
			items[i].Meta["summary"] = anns[i].Summary
		}
		if anns[i].Sentiment != "" {
			items[i].Meta["sentiment"] = strings.ToLower(strings.TrimSpace(anns[i].Sentiment))
		}
	}
}

// runClaudeText shells out to `claude -p` and returns its text output. It mirrors
// ClaudeCodeCurator's error handling: claude writes quota/auth errors to stdout,
// not stderr, so fall back to stdout content when stderr is empty.
func runClaudeText(model string, timeout time.Duration, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{"-p", prompt, "--model", model, "--output-format", "text", "--allowedTools", ""}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = "/tmp"
	cmd.Env = filterEnv("CLAUDECODE")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(string(output))
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("claude -p: %s", msg)
	}
	return string(output), nil
}

var jsonArrRE = regexp.MustCompile(`(?s)\[.*\]`)

func parseAnnotations(output string) ([]Annotation, error) {
	match := jsonArrRE.FindString(output)
	if match == "" {
		return nil, fmt.Errorf("no JSON array found")
	}
	var anns []Annotation
	if err := json.Unmarshal([]byte(match), &anns); err != nil {
		return nil, err
	}
	return anns, nil
}
