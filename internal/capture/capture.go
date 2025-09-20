package capture

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jasperwreed/ai-memory/internal/models"
)

type Capturer struct {
	tool           string
	project        string
	tags           []string
	patternMatcher *PatternMatcher
	tokenEstimator TokenEstimator
}

func NewCapturer(tool, project string, tags []string) *Capturer {
	return &Capturer{
		tool:           tool,
		project:        project,
		tags:           tags,
		patternMatcher: NewPatternMatcher(),
		tokenEstimator: NewSimpleTokenEstimator(),
	}
}

func (c *Capturer) CaptureFromReader(r io.Reader) (*models.Conversation, error) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	content := buf.String()

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("no input to capture")
	}

	if DetectFormat(content) == FormatClaudeCode {
		parser := NewClaudeCodeParser()
		conv, err := parser.ParseJSONL(strings.NewReader(content))
		if err == nil {
			if c.project != "" {
				conv.Project = c.project
			}
			if len(c.tags) > 0 {
				conv.Tags = append(conv.Tags, c.tags...)
			}
			return conv, nil
		}
	}

	lines := strings.Split(content, "\n")
	conversation := c.parseConversation(lines)
	return conversation, nil
}


func (c *Capturer) parseConversation(lines []string) *models.Conversation {
	conv := &models.Conversation{
		Tool:      c.tool,
		Project:   c.project,
		Tags:      c.tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []models.Message{},
	}

	if len(lines) > 0 {
		conv.Title = c.generateTitle(lines)
	}

	messages := c.detectMessages(lines)
	conv.Messages = messages

	return conv
}

func (c *Capturer) detectMessages(lines []string) []models.Message {
	var messages []models.Message
	var currentMessage *models.Message
	var currentContent []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if role, matched := c.patternMatcher.MatchRole(trimmedLine); matched {
			if currentMessage != nil && len(currentContent) > 0 {
				currentMessage.Content = strings.Join(currentContent, "\n")
				messages = append(messages, *currentMessage)
				currentContent = []string{}
			}

			content := c.patternMatcher.ExtractContent(trimmedLine, role)
			currentMessage = &models.Message{
				Role:      string(role),
				Timestamp: time.Now(),
				Content:   content,
			}
			if content != "" {
				currentContent = append(currentContent, content)
			}
		} else if currentMessage != nil && trimmedLine != "" {
			currentContent = append(currentContent, line)
		} else if len(messages) == 0 && trimmedLine != "" {
			currentMessage = &models.Message{
				Role:      string(RoleUser),
				Timestamp: time.Now(),
			}
			currentContent = append(currentContent, line)
		}
	}

	if currentMessage != nil && len(currentContent) > 0 {
		currentMessage.Content = strings.Join(currentContent, "\n")
		messages = append(messages, *currentMessage)
	}

	if len(messages) == 0 && len(lines) > 0 {
		messages = append(messages, models.Message{
			Role:      "user",
			Content:   strings.Join(lines, "\n"),
			Timestamp: time.Now(),
		})
	}

	for i := range messages {
		messages[i].TokenCount = c.tokenEstimator.EstimateTokens(messages[i].Content)
	}

	return messages
}

func (c *Capturer) generateTitle(lines []string) string {
	firstLine := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			firstLine = trimmed
			break
		}
	}

	if firstLine == "" {
		return fmt.Sprintf("%s conversation at %s", c.tool, time.Now().Format("2006-01-02 15:04"))
	}

	prefixes := []string{"Human:", "User:", "You:", "Q:", ">", "Assistant:", "AI:", "Bot:", "A:", "Claude:", "GPT:"}
	for _, prefix := range prefixes {
		firstLine = strings.TrimPrefix(firstLine, prefix)
	}
	firstLine = strings.TrimSpace(firstLine)

	if len(firstLine) > 50 {
		firstLine = firstLine[:50] + "..."
	}

	return firstLine
}


type Parser interface {
	Parse(input string) (*models.Conversation, error)
}

// ClaudeParser handles Claude-specific conversation parsing
type ClaudeParser struct {
	capturer *Capturer
}

// NewClaudeParser creates a new Claude parser
func NewClaudeParser() *ClaudeParser {
	return &ClaudeParser{
		capturer: NewCapturer("claude", "", nil),
	}
}

// Parse parses Claude conversation format
func (p *ClaudeParser) Parse(input string) (*models.Conversation, error) {
	lines := strings.Split(input, "\n")
	return p.capturer.parseConversation(lines), nil
}

// AiderParser handles Aider-specific conversation parsing
type AiderParser struct {
	capturer *Capturer
}

// NewAiderParser creates a new Aider parser
func NewAiderParser() *AiderParser {
	return &AiderParser{
		capturer: NewCapturer("aider", "", nil),
	}
}

// Parse parses Aider conversation format
func (p *AiderParser) Parse(input string) (*models.Conversation, error) {
	lines := strings.Split(input, "\n")
	return p.capturer.parseConversation(lines), nil
}

func DetectToolFromInput(input string) string {
	lowerInput := strings.ToLower(input)

	if strings.Contains(lowerInput, "claude") {
		return "claude"
	}
	if strings.Contains(lowerInput, "aider") {
		return "aider"
	}
	if strings.Contains(lowerInput, "gpt") || strings.Contains(lowerInput, "chatgpt") {
		return "gpt"
	}
	if strings.Contains(lowerInput, "codex") {
		return "codex"
	}

	return "unknown"
}