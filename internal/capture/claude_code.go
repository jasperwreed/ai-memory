package capture

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jasper/ai-memory/internal/models"
)

type ClaudeCodeMessage struct {
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message"`
	Timestamp string          `json:"timestamp"`
	SessionID string          `json:"sessionId"`
	CWD       string          `json:"cwd"`
	Version   string          `json:"version"`
}

type ClaudeUserMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type ClaudeAssistantMessage struct {
	Role    string                   `json:"role"`
	Content []ClaudeContentItem      `json:"content"`
	Model   string                   `json:"model"`
	ID      string                   `json:"id"`
}

type ClaudeContentItem struct {
	Type   string          `json:"type"`
	Text   string          `json:"text,omitempty"`
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
}

type ClaudeToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
}

type ClaudeCodeParser struct{}

func NewClaudeCodeParser() *ClaudeCodeParser {
	return &ClaudeCodeParser{}
}

func (p *ClaudeCodeParser) ParseJSONL(r io.Reader) (*models.Conversation, error) {
	scanner := bufio.NewScanner(r)
	var messages []models.Message
	var sessionID, projectPath string
	var timestamp time.Time

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg ClaudeCodeMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.SessionID != "" && sessionID == "" {
			sessionID = msg.SessionID
		}
		if msg.CWD != "" && projectPath == "" {
			projectPath = msg.CWD
		}

		if msg.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, msg.Timestamp); err == nil {
				timestamp = t
			}
		}

		switch msg.Type {
		case "user":
			if userMsg := p.parseUserMessage(msg.Message); userMsg != nil {
				messages = append(messages, *userMsg)
			}
		case "assistant":
			if assistantMsg := p.parseAssistantMessage(msg.Message); assistantMsg != nil {
				messages = append(messages, *assistantMsg)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading JSONL: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in Claude Code session")
	}

	conv := &models.Conversation{
		Tool:      "claude-code",
		Project:   extractProjectName(projectPath),
		Title:     generateTitleFromMessages(messages),
		CreatedAt: timestamp,
		UpdatedAt: time.Now(),
		Messages:  messages,
		Tags:      []string{"claude-code"},
	}

	return conv, nil
}

func (p *ClaudeCodeParser) parseUserMessage(raw json.RawMessage) *models.Message {
	var userMsg ClaudeUserMessage
	if err := json.Unmarshal(raw, &userMsg); err != nil {
		return nil
	}

	content := ""
	if userMsg.Role == "user" {
		if userMsg.Content != nil {
			var strContent string
			if err := json.Unmarshal(userMsg.Content, &strContent); err == nil {
				content = strContent
			} else {
				var toolResults []ClaudeToolResult
				if err := json.Unmarshal(userMsg.Content, &toolResults); err == nil && len(toolResults) > 0 {
					return nil
				}
			}
		}
	}

	if content == "" {
		return nil
	}

	return &models.Message{
		Role:       "user",
		Content:    content,
		Timestamp:  time.Now(),
		TokenCount: estimateTokens(content),
	}
}

func (p *ClaudeCodeParser) parseAssistantMessage(raw json.RawMessage) *models.Message {
	var assistantMsg ClaudeAssistantMessage
	if err := json.Unmarshal(raw, &assistantMsg); err != nil {
		return nil
	}

	if assistantMsg.Role != "assistant" {
		return nil
	}

	var contentParts []string
	for _, item := range assistantMsg.Content {
		switch item.Type {
		case "text":
			if item.Text != "" {
				contentParts = append(contentParts, item.Text)
			}
		case "tool_use":
			contentParts = append(contentParts, fmt.Sprintf("[Used tool: %s]", item.Name))
		}
	}

	if len(contentParts) == 0 {
		return nil
	}

	content := strings.Join(contentParts, "\n")
	return &models.Message{
		Role:       "assistant",
		Content:    content,
		Timestamp:  time.Now(),
		TokenCount: estimateTokens(content),
	}
}

func extractProjectName(path string) string {
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

func generateTitleFromMessages(messages []models.Message) string {
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			content := strings.TrimSpace(msg.Content)
			if len(content) > 50 {
				content = content[:50] + "..."
			}
			return content
		}
	}
	return fmt.Sprintf("Claude Code session at %s", time.Now().Format("2006-01-02 15:04"))
}

func estimateTokens(text string) int {
	words := strings.Fields(text)
	return len(words) * 4 / 3
}