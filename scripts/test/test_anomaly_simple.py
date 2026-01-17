#!/usr/bin/env python3
"""
Simple test for anomaly detection logic - no external dependencies.
"""
import sys
import os

print("=" * 60)
print("KISS Migration - Anomaly Detection Simple Test")
print("=" * 60)

# Test constants (same as HA component)
ANOMALY_TOLERANCE_PERCENT = 15.0
ANOMALY_TOLERANCE_ABSOLUTE = 0.03
ANOMALY_MIN_PRICE = 0.05
ANOMALY_MAX_PRICE = 1.00


def validate_price_against_reference(enever_price_kwh: float, reference: dict) -> tuple:
    """Validation logic (copy from HA component)."""
    if enever_price_kwh < ANOMALY_MIN_PRICE:
        return (False, f"Price {enever_price_kwh:.4f} below minimum {ANOMALY_MIN_PRICE}")
    if enever_price_kwh > ANOMALY_MAX_PRICE:
        return (False, f"Price {enever_price_kwh:.4f} above maximum {ANOMALY_MAX_PRICE}")

    if not reference or "expected_range" not in reference:
        return (True, "No reference data - accepting BYO price")

    expected_range = reference["expected_range"]
    low = expected_range.get("low", 0)
    high = expected_range.get("high", 1)
    expected = expected_range.get("expected", (low + high) / 2)

    percent_tolerance = expected * (ANOMALY_TOLERANCE_PERCENT / 100)
    tolerance = max(percent_tolerance, ANOMALY_TOLERANCE_ABSOLUTE)

    tolerance_low = low - tolerance
    tolerance_high = high + tolerance

    if tolerance_low <= enever_price_kwh <= tolerance_high:
        return (True, f"Price {enever_price_kwh:.4f} within range [{tolerance_low:.4f}, {tolerance_high:.4f}]")
    else:
        return (False, f"Price {enever_price_kwh:.4f} outside range [{tolerance_low:.4f}, {tolerance_high:.4f}]")


def test_validation():
    """Test validation logic."""
    # Simulated _reference from backend
    reference = {
        "source": "Frank DB",
        "tier": 1,
        "expected_range": {
            "low": 0.20,
            "high": 0.30,
            "expected": 0.25
        }
    }

    print(f"\nReference data:")
    print(f"  expected_range: low={reference['expected_range']['low']}, high={reference['expected_range']['high']}")
    print(f"  Tolerance: {ANOMALY_TOLERANCE_PERCENT}% + €{ANOMALY_TOLERANCE_ABSOLUTE}/kWh")

    # Calculate actual tolerance
    expected = reference["expected_range"]["expected"]
    tolerance = max(expected * (ANOMALY_TOLERANCE_PERCENT / 100), ANOMALY_TOLERANCE_ABSOLUTE)
    tolerance_low = reference["expected_range"]["low"] - tolerance
    tolerance_high = reference["expected_range"]["high"] + tolerance
    print(f"  Effective range: [{tolerance_low:.4f}, {tolerance_high:.4f}]")

    tests = [
        (0.25, True, "Price within range"),
        (0.20, True, "Price at low bound"),
        (0.30, True, "Price at high bound"),
        (tolerance_low + 0.001, True, "Price just inside tolerance low"),
        (tolerance_high - 0.001, True, "Price just inside tolerance high"),
        (0.10, False, "Price way below range (ANOMALY)"),
        (0.50, False, "Price way above range (ANOMALY)"),
        (0.01, False, "Price below minimum"),
        (1.50, False, "Price above maximum"),
    ]

    print("\nRunning tests:")
    all_passed = True

    for price, expected_valid, description in tests:
        is_valid, reason = validate_price_against_reference(price, reference)
        status = "✓" if is_valid == expected_valid else "✗"
        if is_valid != expected_valid:
            all_passed = False
        print(f"  {status} €{price:.4f}/kWh - {description}")
        print(f"      Valid: {is_valid}, Reason: {reason}")

    # Test no reference (graceful degradation)
    print("\n  Testing graceful degradation (no reference):")
    is_valid, reason = validate_price_against_reference(0.25, None)
    status = "✓" if is_valid else "✗"
    if not is_valid:
        all_passed = False
    print(f"  {status} No reference -> should accept price")
    print(f"      Valid: {is_valid}, Reason: {reason}")

    return all_passed


def test_extract_reference():
    """Test reference extraction."""
    print("\nTesting extract_reference_from_server:")

    # Simulate server response
    server_data = {
        "data": [
            {
                "timestamp": "2026-01-17T14:00:00+00:00",
                "price_eur_mwh": 250.0,
                "_reference": {
                    "source": "Frank DB",
                    "tier": 1,
                    "expected_range": {"low": 0.20, "high": 0.30, "expected": 0.25}
                }
            }
        ]
    }

    # Extract reference (same logic as HA component)
    prices = server_data.get("data", [])
    first_price = prices[0] if prices else {}
    reference = first_price.get("_reference")

    if reference and reference["source"] == "Frank DB":
        print("  ✓ Reference extracted correctly")
        print(f"      source: {reference['source']}")
        print(f"      tier: {reference['tier']}")
        print(f"      expected_range: {reference['expected_range']}")
        return True
    else:
        print("  ✗ Failed to extract reference")
        return False


if __name__ == "__main__":
    passed = True

    passed = test_validation() and passed
    passed = test_extract_reference() and passed

    print("\n" + "=" * 60)
    if passed:
        print("✓ ALL TESTS PASSED")
    else:
        print("✗ SOME TESTS FAILED")
    print("=" * 60)

    sys.exit(0 if passed else 1)
