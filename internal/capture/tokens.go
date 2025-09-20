package capture

import "strings"

// TokenEstimator provides methods for estimating token counts
type TokenEstimator interface {
	EstimateTokens(text string) int
}

// SimpleTokenEstimator provides a basic token estimation
type SimpleTokenEstimator struct {
	wordsPerToken float64
}

// NewSimpleTokenEstimator creates a new simple token estimator
// Default estimation is roughly 0.75 words per token (4 tokens per 3 words)
func NewSimpleTokenEstimator() *SimpleTokenEstimator {
	return &SimpleTokenEstimator{
		wordsPerToken: 0.75,
	}
}

// EstimateTokens estimates the number of tokens in the text
func (e *SimpleTokenEstimator) EstimateTokens(text string) int {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	// Estimate: approximately 4 tokens per 3 words
	return (len(words) * 4) / 3
}

// EstimateTokenCost estimates the cost based on token count
func EstimateTokenCost(tokenCount int, costPerToken float64) float64 {
	return float64(tokenCount) * costPerToken
}

// DefaultCostPerToken is a rough estimate for token cost in USD
const DefaultCostPerToken = 0.000003