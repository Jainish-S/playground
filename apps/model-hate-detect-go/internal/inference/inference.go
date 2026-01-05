// Package inference provides hate speech detection logic.
package inference

import "strings"

var hateKeywords = []string{
	"hate", "kill", "die", "attack", "destroy",
	"violence", "violent", "threat", "murder",
	"racist", "sexist", "discriminate", "slur",
	"abuse", "harass", "terror", "extremist",
}

// DetectHateSpeech performs keyword-based hate speech detection.
func DetectHateSpeech(text string) (bool, float64, []string) {
	lowerText := strings.ToLower(text)
	var matches []string

	for _, keyword := range hateKeywords {
		if strings.Contains(lowerText, keyword) {
			matches = append(matches, keyword)
		}
	}

	if len(matches) == 0 {
		return false, 0.0, nil
	}

	score := float64(len(matches)) / float64(len(hateKeywords))
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.5 {
		score = 0.5
	}

	return true, score, []string{"Hate keywords detected: " + strings.Join(matches, ", ")}
}
