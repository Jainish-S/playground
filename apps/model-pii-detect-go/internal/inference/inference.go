// Package inference provides PII detection logic.
package inference

import (
	"regexp"
	"strings"
)

// PII patterns to detect
var (
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phonePattern = regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	ssnPattern   = regexp.MustCompile(`\b\d{3}[-]?\d{2}[-]?\d{4}\b`)
	ccPattern    = regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)
)

var piiKeywords = []string{
	"social security",
	"ssn",
	"credit card",
	"bank account",
	"passport",
	"driver license",
	"date of birth",
	"dob",
}

// DetectPII performs keyword-based PII detection.
func DetectPII(text string) (bool, float64, []string) {
	lowerText := strings.ToLower(text)
	var detections []string

	// Check patterns
	if emailPattern.MatchString(text) {
		detections = append(detections, "email_detected")
	}
	if phonePattern.MatchString(text) {
		detections = append(detections, "phone_detected")
	}
	if ssnPattern.MatchString(text) {
		detections = append(detections, "ssn_pattern_detected")
	}
	if ccPattern.MatchString(text) {
		detections = append(detections, "credit_card_detected")
	}

	// Check keywords
	for _, keyword := range piiKeywords {
		if strings.Contains(lowerText, keyword) {
			detections = append(detections, "keyword:"+keyword)
		}
	}

	if len(detections) == 0 {
		return false, 0.0, nil
	}

	score := float64(len(detections)) * 0.2
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.5 {
		score = 0.5
	}

	return true, score, detections
}
