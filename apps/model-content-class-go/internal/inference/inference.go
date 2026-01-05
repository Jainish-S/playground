// Package inference provides content classification logic.
package inference

import "strings"

var (
	violenceKeywords = []string{"violence", "fight", "weapon", "gun", "blood", "injury", "assault"}
	adultKeywords    = []string{"adult", "explicit", "nsfw", "nude", "sexual"}
	spamKeywords     = []string{"buy now", "click here", "free", "winner", "prize", "urgent", "act now"}
	drugKeywords     = []string{"drug", "cocaine", "heroin", "meth", "opioid"}
)

// ClassifyContent performs keyword-based content classification.
func ClassifyContent(text string) (bool, float64, []string) {
	lowerText := strings.ToLower(text)
	var categories []string

	for _, k := range violenceKeywords {
		if strings.Contains(lowerText, k) {
			categories = append(categories, "violence")
			break
		}
	}
	for _, k := range adultKeywords {
		if strings.Contains(lowerText, k) {
			categories = append(categories, "adult")
			break
		}
	}
	for _, k := range spamKeywords {
		if strings.Contains(lowerText, k) {
			categories = append(categories, "spam")
			break
		}
	}
	for _, k := range drugKeywords {
		if strings.Contains(lowerText, k) {
			categories = append(categories, "drugs")
			break
		}
	}

	if len(categories) == 0 {
		return false, 0.0, nil
	}

	score := float64(len(categories)) * 0.25
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.5 {
		score = 0.5
	}

	return true, score, []string{"Flagged categories: " + strings.Join(categories, ", ")}
}
