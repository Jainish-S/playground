// Package inference provides the prompt injection detection logic.
package inference

import (
	"strings"
)

// Injection keywords to detect
var injectionKeywords = []string{
	"ignore",
	"ignore previous",
	"disregard",
	"forget",
	"bypass",
	"override",
	"reveal",
	"expose",
	"jailbreak",
	"pretend",
	"roleplay",
	"act as",
	"you are now",
	"ignore all",
	"system prompt",
	"instructions",
	"secret",
	"password",
	"confidential",
}

// DetectPromptInjection performs keyword-based prompt injection detection.
// This is a dummy implementation that simulates real ML inference.
//
// Returns:
//   - flagged: true if injection detected
//   - score: confidence score (0.0-1.0)
//   - details: list of reasons
func DetectPromptInjection(text string) (bool, float64, []string) {
	lowerText := strings.ToLower(text)
	var matchedKeywords []string

	for _, keyword := range injectionKeywords {
		if strings.Contains(lowerText, keyword) {
			matchedKeywords = append(matchedKeywords, keyword)
		}
	}

	if len(matchedKeywords) == 0 {
		return false, 0.0, nil
	}

	// Calculate score based on number of matches
	score := float64(len(matchedKeywords)) / float64(len(injectionKeywords))
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.5 {
		score = 0.5 // Minimum confidence for flagged content
	}

	details := []string{"Injection keywords detected: " + strings.Join(matchedKeywords, ", ")}

	return true, score, details
}
