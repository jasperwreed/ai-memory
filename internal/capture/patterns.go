package capture

import (
	"regexp"
	"strings"
)

// MessageRole represents the role in a conversation
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

// PatternMatcher handles pattern matching for message detection
type PatternMatcher struct {
	userPatterns      *regexp.Regexp
	assistantPatterns *regexp.Regexp
}

// NewPatternMatcher creates a new pattern matcher with default patterns
func NewPatternMatcher() *PatternMatcher {
	return &PatternMatcher{
		userPatterns:      regexp.MustCompile(`^(Human:|User:|You:|Q:|>)`),
		assistantPatterns: regexp.MustCompile(`^(Assistant:|AI:|Bot:|A:|Claude:|GPT:)`),
	}
}

// MatchRole determines the role based on the line content
func (pm *PatternMatcher) MatchRole(line string) (MessageRole, bool) {
	trimmedLine := strings.TrimSpace(line)

	if pm.userPatterns.MatchString(trimmedLine) {
		return RoleUser, true
	}

	if pm.assistantPatterns.MatchString(trimmedLine) {
		return RoleAssistant, true
	}

	return "", false
}

// ExtractContent removes the role prefix from the line
func (pm *PatternMatcher) ExtractContent(line string, role MessageRole) string {
	trimmedLine := strings.TrimSpace(line)

	switch role {
	case RoleUser:
		return strings.TrimSpace(pm.userPatterns.ReplaceAllString(trimmedLine, ""))
	case RoleAssistant:
		return strings.TrimSpace(pm.assistantPatterns.ReplaceAllString(trimmedLine, ""))
	default:
		return trimmedLine
	}
}

// DetectFormat checks if the content is in Claude Code JSONL format
func DetectFormat(content string) FormatType {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return FormatPlainText
	}

	firstLine := strings.TrimSpace(lines[0])
	if strings.Contains(firstLine, `"sessionId"`) &&
		strings.Contains(firstLine, `"type"`) &&
		(strings.Contains(firstLine, `"user"`) || strings.Contains(firstLine, `"assistant"`)) {
		return FormatClaudeCode
	}

	return FormatPlainText
}

// FormatType represents the format of the input content
type FormatType int

const (
	FormatPlainText FormatType = iota
	FormatClaudeCode
)