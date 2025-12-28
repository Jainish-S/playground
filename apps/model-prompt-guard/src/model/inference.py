"""Dummy prompt injection detection.

This detects common prompt injection patterns using keyword matching.
Will be replaced with actual ML model later.
"""

import re

# Common prompt injection patterns
INJECTION_PATTERNS = [
    r"ignore\s+(previous|all|any)\s+(instructions?|prompts?)",
    r"disregard\s+(previous|all|any)",
    r"forget\s+(everything|previous|all)",
    r"you\s+are\s+now",
    r"new\s+instructions?",
    r"jailbreak",
    r"bypass\s+(safety|filters?|restrictions?)",
    r"reveal\s+(secrets?|hidden|system)",
    r"system\s+prompt",
    r"(\[|\{)\s*(system|SYSTEM)",
    r"pretend\s+(you\s+are|to\s+be)",
    r"act\s+as\s+(if|a)",
    r"roleplay",
    r"DAN\s+mode",
]

COMPILED_PATTERNS = [re.compile(p, re.IGNORECASE) for p in INJECTION_PATTERNS]


def detect_prompt_injection(text: str) -> tuple[bool, float, list[str]]:
    """
    Detect prompt injection attempts in text.
    
    Returns:
        Tuple of (flagged, score, details)
    """
    matches = []
    for pattern in COMPILED_PATTERNS:
        if pattern.search(text):
            matches.append(f"Pattern matched: {pattern.pattern}")
    
    if matches:
        # Score based on number of matches
        score = min(0.5 + (len(matches) * 0.15), 1.0)
        return True, score, matches
    
    return False, 0.0, []
