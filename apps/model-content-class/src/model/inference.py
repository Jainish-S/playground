"""Dummy content classification.

Classifies content into policy categories using keyword matching.
Will be replaced with zero-shot classifier later.
"""

import random

# Content categories for classification
CATEGORIES = {
    "legal": ["lawsuit", "court", "legal", "attorney", "contract", "litigation"],
    "financial": ["invest", "stock", "money", "bank", "credit", "loan", "price"],
    "medical": ["health", "doctor", "medicine", "symptom", "disease", "treatment"],
    "adult": ["explicit", "nsfw", "adult", "mature"],
    "violence": ["fight", "weapon", "gun", "knife", "attack", "war"],
    "political": ["election", "vote", "democrat", "republican", "politics", "government"],
}

# Categories that trigger flagging
FLAGGED_CATEGORIES = {"adult", "violence"}


def classify_content(text: str) -> tuple[bool, float, list[str]]:
    """
    Classify content into categories.
    
    Returns:
        Tuple of (flagged, score, details)
    """
    text_lower = text.lower()
    detected_categories = []
    
    for category, keywords in CATEGORIES.items():
        if any(kw in text_lower for kw in keywords):
            detected_categories.append(category)
    
    # If no categories detected, assign a random safe one
    if not detected_categories:
        detected_categories = [random.choice(["general", "casual", "informational"])]
    
    # Check if any flagged category is detected
    flagged_cats = [c for c in detected_categories if c in FLAGGED_CATEGORIES]
    
    if flagged_cats:
        score = min(0.6 + (len(flagged_cats) * 0.2), 1.0)
        details = [f"Flagged category: {cat}" for cat in flagged_cats]
        details.extend([f"Category: {cat}" for cat in detected_categories if cat not in FLAGGED_CATEGORIES])
        return True, score, details
    
    return False, 0.1, [f"Category: {cat}" for cat in detected_categories]
