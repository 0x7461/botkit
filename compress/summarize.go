package compress

import (
	"bytes"
	"os/exec"
	"strings"
)

// Summarize runs the summarize CLI on a URL and returns the first paragraph
// as a compact description. Returns empty string if the CLI is unavailable or fails.
func Summarize(url string) string {
	cmd := exec.Command("summarize", url, "--cli", "claude", "--plain", "--length", "short")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return firstParagraph(out.String())
}

// firstParagraph extracts the first non-empty block of text before a blank line.
func firstParagraph(s string) string {
	s = strings.TrimSpace(s)
	var para []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(para) > 0 {
				break
			}
			continue
		}
		para = append(para, trimmed)
	}
	return strings.Join(para, " ")
}
