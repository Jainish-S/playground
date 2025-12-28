"""Dummy PII detection.

Detects common PII patterns using regex.
Will be replaced with Presidio/ML model later.
"""

import re
from typing import NamedTuple


class PIIMatch(NamedTuple):
    type: str
    value: str


# PII patterns
PII_PATTERNS = {
    "email": r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b",
    "phone_us": r"\b(\+1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b",
    "ssn": r"\b\d{3}[-.\s]?\d{2}[-.\s]?\d{4}\b",
    "credit_card": r"\b(?:\d{4}[-.\s]?){3}\d{4}\b",
    "ip_address": r"\b(?:\d{1,3}\.){3}\d{1,3}\b",
    "date_of_birth": r"\b(?:0[1-9]|1[0-2])[/-](?:0[1-9]|[12]\d|3[01])[/-](?:19|20)\d{2}\b",
}

COMPILED_PATTERNS = {name: re.compile(pattern) for name, pattern in PII_PATTERNS.items()}


def detect_pii(text: str) -> tuple[bool, float, list[str]]:
    """
    Detect PII in text.
    
    Returns:
        Tuple of (flagged, score, details)
    """
    matches = []
    
    for pii_type, pattern in COMPILED_PATTERNS.items():
        found = pattern.findall(text)
        if found:
            # Redact the actual values
            for match in found[:3]:  # Limit to first 3 matches per type
                redacted = match[:2] + "***" + match[-2:] if len(match) > 4 else "***"
                matches.append(f"{pii_type}: {redacted}")
    
    if matches:
        # Score based on number and types of PII
        score = min(0.6 + (len(matches) * 0.1), 1.0)
        return True, score, matches
    
    return False, 0.0, []
