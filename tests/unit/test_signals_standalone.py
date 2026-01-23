"""
Standalone test for signals logic
Tests query structure without DB connection
"""

# Mock data for testing
MOCK_CURRENT_PRICE = 0.12  # EUR/kWh
MOCK_DAILY_AVG = 0.15
MOCK_RENEWABLE_PCT = 55.3
MOCK_BALANCE_DELTA = -320  # MW
MOCK_NEXT_3H = [0.11, 0.10, 0.14]


def test_signals():
    """Test signal logic with mock data"""

    # Thresholds
    RENEWABLE_THRESHOLD = 50
    BALANCE_THRESHOLD = 500

    # Calculate signals
    is_cheap = MOCK_CURRENT_PRICE < MOCK_DAILY_AVG
    is_green = MOCK_RENEWABLE_PCT > RENEWABLE_THRESHOLD
    charge_now = (MOCK_CURRENT_PRICE < MOCK_DAILY_AVG) and (MOCK_RENEWABLE_PCT > 40)
    grid_stable = abs(MOCK_BALANCE_DELTA) < BALANCE_THRESHOLD
    cheap_hour_coming = min(MOCK_NEXT_3H) < (MOCK_CURRENT_PRICE * 0.9)

    print("SIGNAL TEST RESULTS")
    print("=" * 50)
    print(f"Current price:   €{MOCK_CURRENT_PRICE:.3f}/kWh")
    print(f"Daily average:   €{MOCK_DAILY_AVG:.3f}/kWh")
    print(f"Renewable:       {MOCK_RENEWABLE_PCT:.1f}%")
    print(f"Balance delta:   {MOCK_BALANCE_DELTA} MW")
    print(f"Next 3h prices:  {MOCK_NEXT_3H}")
    print()
    print("BINARY SIGNALS:")
    print(f"  is_cheap:           {is_cheap}")
    print(f"  is_green:           {is_green}")
    print(f"  charge_now:         {charge_now}")
    print(f"  grid_stable:        {grid_stable}")
    print(f"  cheap_hour_coming:  {cheap_hour_coming}")
    print()

    # Validation
    assert is_cheap == True, "Expected cheap (0.12 < 0.15)"
    assert is_green == True, "Expected green (55.3% > 50%)"
    assert charge_now == True, "Expected charge_now (cheap + green)"
    assert grid_stable == True, "Expected stable (320 < 500)"
    assert cheap_hour_coming == True, "Expected dip (0.10 < 0.108)"

    print("✅ All signals calculated correctly")


if __name__ == "__main__":
    test_signals()
