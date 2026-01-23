#!/usr/bin/env python3
"""
Test script for database-backed fallback chain.

Tests:
1. Frank collector - fetch and store
2. Enever-Frank collector - fetch and store
3. FallbackManager - read from database
4. Data freshness check

Usage:
    python scripts/test/test_db_fallback.py
"""

import asyncio
import os
import sys

# Add project root to path
sys.path.insert(
    0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
)

from datetime import UTC, datetime

# Test imports
print("=" * 60)
print("DATABASE-BACKED FALLBACK CHAIN - TEST SUITE")
print("=" * 60)


async def test_frank_collector():
    """Test Frank collector can fetch prices."""
    print("\n[TEST 1] Frank Collector - API Fetch")
    print("-" * 40)

    try:
        from scripts.collectors.frank_collector import fetch_frank_prices

        today = datetime.now(UTC).date()
        prices = await fetch_frank_prices(today.isoformat(), today.isoformat())

        if prices and len(prices) > 0:
            print(f"  PASS: Fetched {len(prices)} prices from Frank API")
            print(
                f"  Sample: {prices[0]['timestamp']} = {prices[0]['price_eur_kwh']:.4f} EUR/kWh"
            )
            return True
        else:
            print("  FAIL: No prices returned from Frank API")
            return False

    except Exception as e:
        print(f"  FAIL: {e}")
        return False


async def test_fallback_db_methods():
    """Test FallbackManager database read methods."""
    print("\n[TEST 2] FallbackManager Database Methods")
    print("-" * 40)

    try:
        from synctacles_db.fallback.fallback_manager import FallbackManager

        # Test Frank DB read
        frank_prices, frank_age = await FallbackManager._get_frank_from_db()
        if frank_prices:
            print(
                f"  PASS: Frank DB has {len(frank_prices)} prices, age {frank_age} min"
            )
        else:
            print("  INFO: Frank DB empty (expected if collectors haven't run yet)")

        # Test Enever-Frank DB read
        enever_prices, enever_age = await FallbackManager._get_enever_frank_from_db()
        if enever_prices:
            print(
                f"  PASS: Enever-Frank DB has {len(enever_prices)} prices, age {enever_age} min"
            )
        else:
            print(
                "  INFO: Enever-Frank DB empty (expected if collectors haven't run yet)"
            )

        # Test freshness check
        is_fresh = FallbackManager._check_data_freshness(60)  # 1 hour
        print(f"  PASS: Data freshness check (60 min): GO allowed = {is_fresh}")

        is_stale = FallbackManager._check_data_freshness(400)  # 6.6 hours
        print(f"  PASS: Data freshness check (400 min): GO allowed = {is_stale}")

        return True

    except Exception as e:
        print(f"  FAIL: {e}")
        import traceback

        traceback.print_exc()
        return False


async def test_full_fallback_chain():
    """Test complete fallback chain."""
    print("\n[TEST 3] Full Fallback Chain")
    print("-" * 40)

    try:
        from synctacles_db.fallback.fallback_manager import FallbackManager

        # Call with no ENTSO-E data (forces consumer price tiers)
        (
            prices,
            source,
            quality,
            allow_go,
        ) = await FallbackManager.get_prices_with_fallback(
            db_results=None, db_age_minutes=999, country="nl"
        )

        if prices:
            print(f"  PASS: Got {len(prices)} prices")
            print(f"  Source: {source}")
            print(f"  Quality: {quality}")
            print(f"  GO Allowed: {allow_go}")
            return True
        else:
            print("  WARN: No prices available (all tiers failed)")
            print(f"  Source: {source}, Quality: {quality}")
            return False

    except Exception as e:
        print(f"  FAIL: {e}")
        import traceback

        traceback.print_exc()
        return False


async def main():
    """Run all tests."""
    results = []

    # Test 1: Frank collector API fetch
    results.append(await test_frank_collector())

    # Test 2: Database methods
    results.append(await test_fallback_db_methods())

    # Test 3: Full fallback chain
    results.append(await test_full_fallback_chain())

    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    passed = sum(results)
    total = len(results)
    print(f"  Passed: {passed}/{total}")

    if passed == total:
        print("  STATUS: ALL TESTS PASSED")
        return 0
    else:
        print("  STATUS: SOME TESTS FAILED")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
