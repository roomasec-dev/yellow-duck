package textutil

import (
	"regexp"
	"strings"
)

var systemReminderRE = regexp.MustCompile(`(?is)<system-reminder>.*?</system-reminder>`)
var angleTagRE = regexp.MustCompile(`(?is)</?system-reminder[^>]*>`)
var reminderLineRE = regexp.MustCompile(`(?im)^\s*(your operational mode has changed from plan to build\.?|you are no longer in read-only mode\.?|you are permitted to make file changes, run shell commands, and utilize your arsenal of tools as needed\.?)\s*$`)
var markdownNumberRE = regexp.MustCompile(`^\d+\.\s+`)

func SanitizeReply(text string) string {
	text = systemReminderRE.ReplaceAllString(text, "")
	text = angleTagRE.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "Your operational mode has changed from plan to build.", "")
	text = strings.ReplaceAll(text, "You are no longer in read-only mode.", "")
	text = strings.ReplaceAll(text, "You are permitted to make file changes, run shell commands, and utilize your arsenal of tools as needed.", "")
	text = reminderLineRE.ReplaceAllString(text, "")

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !blank {
				cleaned = append(cleaned, "")
			}
			blank = true
			continue
		}
		blank = false
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "- "):
			line = "• " + strings.TrimSpace(line[2:])
		case strings.HasPrefix(line, "* "):
			line = "• " + strings.TrimSpace(line[2:])
		case markdownNumberRE.MatchString(line):
			idx := strings.Index(line, ".")
			line = line[:idx] + ") " + strings.TrimSpace(line[idx+1:])
		}
		cleaned = append(cleaned, line)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
