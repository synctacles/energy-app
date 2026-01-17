#!/usr/bin/env python3
"""
End-to-end test for anomaly detection between Backend and HA Component.

Tests:
1. Backend returns _reference data with expected_range
2. HA component validation logic works correctly
3. Anomaly detection triggers fallback when price is out of range
"""
import asyncio
import sys
import os
from datetime import datetime, timezone

# Add project paths
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
sys.path.insert(0, "/opt/github/ha-energy-insights-nl")

print("=" * 60)
print("KISS Migration - Anomaly Detection E2E Test")
print("=" * 60)


async def test_backend_reference_data():
    """Test that backend returns _reference data."""
    print("\n[1] Testing Backend _reference data generation...")

    from synctacles_db.fallback.fallback_manager import FallbackManager
    from synctacles_db.config.static_offsets import HOURLY_OFFSET, get_expected_range, get_market_stats

    # Test static offsets
    print(f"    Static offsets loaded: {len(HOURLY_OFFSET)} hours")
    assert len(HOURLY_OFFSET) == 24, "Should have 24 hourly offsets"

    # Test market stats
    test_prices = [0.20, 0.25, 0.30, 0.15, 0.22]
    stats = get_market_stats(test_prices)
    print(f"    Market stats: avg={stats['average']:.4f}, min={stats['min']:.4f}, max={stats['max']:.4f}")
    assert stats is not None, "Market stats should not be None"

    # Test expected range
    current_hour = datetime.now(timezone.utc).hour
    expected = get_expected_range(0.05, current_hour)  # €0.05/kWh wholesale
    print(f"    Expected range (hour {current_hour}): low={expected['low']:.4f}, high={expected['high']:.4f}")
    assert "low" in expected and "high" in expected, "Expected range should have low/high"

    # Test _add_reference_data
    test_prices_mwh = [
        {"timestamp": datetime.now(timezone.utc).isoformat(), "price_eur_mwh": 250.0},
        {"timestamp": datetime.now(timezone.utc).isoformat(), "price_eur_mwh": 280.0},
    ]
    result = FallbackManager._add_reference_data(test_prices_mwh, "Test Source", 1)

    assert "_reference" in result[0], "First price should have _reference"
    ref = result[0]["_reference"]
    print(f"    _reference added: source={ref['source']}, tier={ref['tier']}")
    print(f"    expected_range: {ref['expected_range']}")

    assert "expected_range" in ref, "_reference should have expected_range"
    assert "low" in ref["expected_range"], "expected_range should have low"
    assert "high" in ref["expected_range"], "expected_range should have high"

    print("    ✓ Backend _reference data generation OK")
    return ref


async def test_ha_validation_logic(reference):
    """Test HA component validation logic."""
    print("\n[2] Testing HA Component validation logic...")

    # Import HA component functions
    from custom_components.ha_energy_insights_nl.sensor import (
        validate_price_against_reference,
        extract_reference_from_server,
    )
    from custom_components.ha_energy_insights_nl.const import (
        ANOMALY_TOLERANCE_PERCENT,
        ANOMALY_TOLERANCE_ABSOLUTE,
        ANOMALY_MIN_PRICE,
        ANOMALY_MAX_PRICE,
    )

    print(f"    Tolerance: {ANOMALY_TOLERANCE_PERCENT}% + €{ANOMALY_TOLERANCE_ABSOLUTE}/kWh")
    print(f"    Bounds: €{ANOMALY_MIN_PRICE} - €{ANOMALY_MAX_PRICE}/kWh")

    expected_range = reference["expected_range"]
    expected_price = expected_range["expected"]

    # Test 1: Price within range should pass
    print("\n    Test 1: Price within range...")
    is_valid, reason = validate_price_against_reference(expected_price, reference)
    print(f"    Price: €{expected_price:.4f}/kWh -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert is_valid, f"Expected price should be valid: {reason}"
    print("    ✓ Pass")

    # Test 2: Price slightly above range (within tolerance) should pass
    print("\n    Test 2: Price slightly above range (within tolerance)...")
    tolerance = max(expected_price * (ANOMALY_TOLERANCE_PERCENT / 100), ANOMALY_TOLERANCE_ABSOLUTE)
    price_above = expected_range["high"] + tolerance - 0.001
    is_valid, reason = validate_price_against_reference(price_above, reference)
    print(f"    Price: €{price_above:.4f}/kWh -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert is_valid, f"Price within tolerance should be valid: {reason}"
    print("    ✓ Pass")

    # Test 3: Price way above range should fail (ANOMALY)
    print("\n    Test 3: Price way above range (ANOMALY)...")
    price_anomaly = expected_range["high"] + 0.20  # €0.20 above high
    is_valid, reason = validate_price_against_reference(price_anomaly, reference)
    print(f"    Price: €{price_anomaly:.4f}/kWh -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert not is_valid, f"Anomaly price should be invalid: {reason}"
    print("    ✓ Pass (correctly detected anomaly)")

    # Test 4: Price below minimum should fail
    print("\n    Test 4: Price below minimum...")
    is_valid, reason = validate_price_against_reference(0.01, reference)
    print(f"    Price: €0.01/kWh -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert not is_valid, "Price below minimum should be invalid"
    print("    ✓ Pass")

    # Test 5: Price above maximum should fail
    print("\n    Test 5: Price above maximum...")
    is_valid, reason = validate_price_against_reference(1.50, reference)
    print(f"    Price: €1.50/kWh -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert not is_valid, "Price above maximum should be invalid"
    print("    ✓ Pass")

    # Test 6: No reference data - graceful degradation
    print("\n    Test 6: No reference data (graceful degradation)...")
    is_valid, reason = validate_price_against_reference(0.25, None)
    print(f"    Price: €0.25/kWh, No reference -> Valid: {is_valid}")
    print(f"    Reason: {reason}")
    assert is_valid, "Should accept price when no reference available"
    print("    ✓ Pass")

    print("\n    ✓ HA Component validation logic OK")


async def test_extract_reference():
    """Test extracting reference from server response."""
    print("\n[3] Testing extract_reference_from_server...")

    from custom_components.ha_energy_insights_nl.sensor import extract_reference_from_server

    # Simulate server response with _reference
    server_data = {
        "data": [
            {
                "timestamp": "2026-01-17T14:00:00+00:00",
                "price_eur_mwh": 250.0,
                "_reference": {
                    "source": "Frank DB",
                    "tier": 1,
                    "expected_range": {
                        "low": 0.18,
                        "high": 0.32,
                        "expected": 0.25
                    }
                }
            },
            {
                "timestamp": "2026-01-17T15:00:00+00:00",
                "price_eur_mwh": 260.0
            }
        ]
    }

    reference = extract_reference_from_server(server_data)
    print(f"    Extracted reference: {reference}")
    assert reference is not None, "Should extract reference"
    assert reference["source"] == "Frank DB", "Source should match"
    assert reference["tier"] == 1, "Tier should match"
    print("    ✓ Extract reference OK")

    # Test empty response
    reference = extract_reference_from_server({})
    assert reference is None, "Should return None for empty response"
    print("    ✓ Empty response handling OK")

    # Test response without _reference
    server_data_no_ref = {"data": [{"timestamp": "2026-01-17T14:00:00+00:00", "price_eur_mwh": 250.0}]}
    reference = extract_reference_from_server(server_data_no_ref)
    assert reference is None, "Should return None when no _reference"
    print("    ✓ Missing _reference handling OK")


async def main():
    """Run all tests."""
    try:
        # Test backend
        reference = await test_backend_reference_data()

        # Test HA validation
        await test_ha_validation_logic(reference)

        # Test reference extraction
        await test_extract_reference()

        print("\n" + "=" * 60)
        print("✓ ALL TESTS PASSED - Anomaly Detection E2E OK")
        print("=" * 60)
        return 0

    except AssertionError as e:
        print(f"\n✗ TEST FAILED: {e}")
        return 1
    except Exception as e:
        print(f"\n✗ ERROR: {e}")
        import traceback
        traceback.print_exc()
        return 2


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
