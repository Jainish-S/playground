"""Dummy hate speech detection.

Detects hate speech and toxic content using keyword matching.
Will be replaced with fine-tuned BERT model later.
"""

import re

# Simplified toxic content patterns (placeholder - not comprehensive)
# In production, use a proper ML model
TOXIC_KEYWORDS = [
    "hate",
    "kill",
    "attack",
    "destroy",
    "violent",
    "threat",
    "harm",
    "hurt",
]

# Patterns that indicate targeted harassment
HARASSMENT_PATTERNS = [
    r"you\s+(are|should)\s+(die|dead|kill)",
    r"go\s+die",
    r"i\s+will\s+(kill|hurt|attack)",
    r"you\s+deserve\s+(death|pain|suffering)",
]

COMPILED_PATTERNS = [re.compile(p, re.IGNORECASE) for p in HARASSMENT_PATTERNS]


def detect_hate(text: str) -> tuple[bool, float, list[str]]:
    """
    Detect hate speech in text.
    
    Returns:
        Tuple of (flagged, score, details)
    """
    text_lower = text.lower()
    details = []
    
    # Check harassment patterns
    for pattern in COMPILED_PATTERNS:
        if pattern.search(text):
            details.append(f"Pattern matched: harassment/threat")
            break  # One match is enough
    
    # Count toxic keywords (simple approach)
    keyword_count = sum(1 for kw in TOXIC_KEYWORDS if kw in text_lower)
    
    if keyword_count >= 2:
        details.append(f"Multiple toxic keywords detected ({keyword_count})")
    
    if details:
        score = min(0.5 + (len(details) * 0.2) + (keyword_count * 0.05), 1.0)
        return True, score, details
    
    return False, 0.0, []
