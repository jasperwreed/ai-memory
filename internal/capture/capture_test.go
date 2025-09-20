package capture

import (
	"strings"
	"testing"
)

func TestCaptureFromReader(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedMsgs  int
		expectedTitle string
		tool          string
	}{
		{
			name: "Simple Q&A format",
			input: `User: How do I implement a binary search?
Assistant: To implement binary search, you need a sorted array and divide the search space in half repeatedly.`,
			expectedMsgs:  2,
			expectedTitle: "How do I implement a binary search?",
			tool:          "test",
		},
		{
			name: "Claude format",
			input: `Human: What is recursion?
Assistant: Recursion is a programming technique where a function calls itself.`,
			expectedMsgs:  2,
			expectedTitle: "What is recursion?",
			tool:          "claude",
		},
		{
			name: "Multiline messages",
			input: `User: Explain Docker
Assistant: Docker is a containerization platform that:
- Packages applications
- Ensures consistency
- Simplifies deployment`,
			expectedMsgs:  2,
			expectedTitle: "Explain Docker",
			tool:          "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturer := NewCapturer(tt.tool, "test-project", []string{"test"})
			reader := strings.NewReader(tt.input)

			conv, err := capturer.CaptureFromReader(reader)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(conv.Messages) != tt.expectedMsgs {
				t.Errorf("expected %d messages, got %d", tt.expectedMsgs, len(conv.Messages))
			}

			if conv.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, conv.Title)
			}

			if conv.Tool != tt.tool {
				t.Errorf("expected tool %q, got %q", tt.tool, conv.Tool)
			}
		})
	}
}

func TestDetectToolFromInput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Using Claude for code generation", "claude"},
		{"Aider helped me fix the bug", "aider"},
		{"ChatGPT response was helpful", "gpt"},
		{"Codex generated this function", "codex"},
		{"Random text without tool mention", "unknown"},
	}

	for _, tt := range tests {
		result := DetectToolFromInput(tt.input)
		if result != tt.expected {
			t.Errorf("DetectToolFromInput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	estimator := NewSimpleTokenEstimator()

	tests := []struct {
		text     string
		minTokens int
		maxTokens int
	}{
		{"Hello world", 2, 4},
		{"This is a longer sentence with more words", 8, 12},
		{"", 0, 0},
	}

	for _, tt := range tests {
		tokens := estimator.EstimateTokens(tt.text)
		if tokens < tt.minTokens || tokens > tt.maxTokens {
			t.Errorf("EstimateTokens(%q) = %d, want between %d and %d",
				tt.text, tokens, tt.minTokens, tt.maxTokens)
		}
	}
}