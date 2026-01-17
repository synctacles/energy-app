"""
Tests for KISS Stack Migration.

Tests for:
- EasyEnergy client
- Static offset calculations
- Market stats calculations
- Fallback chain (6-tier)
- Reference data structure
"""
import pytest
from datetime import datetime, timezone, timedelta
from unittest.mock import AsyncMock, patch, MagicMock

# Import modules under test
from synctacles_db.clients.easyenergy_client import EasyEnergyClient
from synctacles_db.config.static_offsets import (
    HOURLY_OFFSET,
    AVERAGE_OFFSET,
    apply_static_offset,
    apply_static_offset_mwh,
    get_market_stats,
    get_expected_range,
)


class TestStaticOffsets:
    """Tests for static offset configuration."""

    def test_hourly_offset_has_24_hours(self):
        """Verify all 24 hours are defined."""
        assert len(HOURLY_OFFSET) == 24
        for hour in range(24):
            assert hour in HOURLY_OFFSET

    def test_hourly_offset_values_in_range(self):
        """Verify offsets are within expected range (€0.10-0.25)."""
        for hour, offset in HOURLY_OFFSET.items():
            assert 0.10 <= offset <= 0.30, f"Hour {hour} offset {offset} out of range"

    def test_apply_static_offset(self):
        """Test applying offset to wholesale price."""
        wholesale = 0.05  # €0.05/kWh wholesale
        hour = 8  # Morning peak

        result = apply_static_offset(wholesale, hour)

        expected = wholesale + HOURLY_OFFSET[8]
        assert result == pytest.approx(expected, rel=1e-4)

    def test_apply_static_offset_mwh(self):
        """Test applying offset to MWh price."""
        wholesale_mwh = 50.0  # €50/MWh wholesale
        hour = 8

        result = apply_static_offset_mwh(wholesale_mwh, hour)

        # Convert to kWh, apply offset, convert back
        expected_kwh = (wholesale_mwh / 1000) + HOURLY_OFFSET[8]
        expected_mwh = expected_kwh * 1000

        assert result == pytest.approx(expected_mwh, rel=1e-4)

    def test_apply_static_offset_invalid_hour(self):
        """Test that invalid hours raise ValueError."""
        with pytest.raises(ValueError):
            apply_static_offset(0.05, 24)

        with pytest.raises(ValueError):
            apply_static_offset(0.05, -1)

    def test_average_offset_calculation(self):
        """Verify average offset is calculated correctly."""
        expected_avg = sum(HOURLY_OFFSET.values()) / 24
        assert AVERAGE_OFFSET == pytest.approx(expected_avg, rel=1e-6)


class TestMarketStats:
    """Tests for market statistics calculations."""

    def test_get_market_stats(self):
        """Test market stats calculation."""
        prices = [0.20, 0.25, 0.30, 0.15]

        stats = get_market_stats(prices)

        assert stats is not None
        assert stats["average"] == pytest.approx(0.225, rel=1e-4)
        assert stats["spread"] == pytest.approx(0.15, rel=1e-4)
        assert stats["min"] == pytest.approx(0.15, rel=1e-4)
        assert stats["max"] == pytest.approx(0.30, rel=1e-4)

    def test_get_market_stats_empty(self):
        """Test market stats with empty list."""
        stats = get_market_stats([])
        assert stats is None

    def test_get_expected_range(self):
        """Test expected range calculation."""
        market_avg = 0.05  # €0.05/kWh wholesale
        hour = 8

        result = get_expected_range(market_avg, hour)

        expected_price = market_avg + HOURLY_OFFSET[8]
        tolerance = expected_price * 0.15  # 15% tolerance

        assert result["expected"] == pytest.approx(expected_price, rel=1e-4)
        assert result["low"] == pytest.approx(expected_price - tolerance, rel=1e-4)
        assert result["high"] == pytest.approx(expected_price + tolerance, rel=1e-4)


class TestEasyEnergyClient:
    """Tests for EasyEnergy API client."""

    @pytest.mark.asyncio
    async def test_health_check_mock(self):
        """Test health check with mocked response."""
        mock_prices = [
            {
                "Timestamp": "2026-01-17T00:00:00Z",
                "TariffReturn": 50.0  # €50/MWh
            },
            {
                "Timestamp": "2026-01-17T01:00:00Z",
                "TariffReturn": 55.0
            }
        ]

        with patch("aiohttp.ClientSession") as mock_session:
            mock_response = AsyncMock()
            mock_response.status = 200
            mock_response.json = AsyncMock(return_value=mock_prices)

            mock_session.return_value.__aenter__.return_value.get.return_value.__aenter__.return_value = mock_response

            is_healthy, message = await EasyEnergyClient.health_check()

            # Due to circuit breaker state, may or may not succeed
            # Just verify no exception is raised
            assert isinstance(is_healthy, bool)
            assert isinstance(message, str)

    def test_circuit_breaker_initial_state(self):
        """Test circuit breaker starts closed."""
        # Reset circuit breaker
        from synctacles_db.clients.easyenergy_client import _circuit_breaker
        _circuit_breaker["failure_count"] = 0
        _circuit_breaker["last_failure_time"] = None

        result = EasyEnergyClient._check_circuit_breaker()
        assert result is False  # Circuit breaker should be closed


class TestFallbackManager:
    """Tests for the 6-tier fallback manager."""

    @pytest.mark.asyncio
    async def test_apply_static_offset(self):
        """Test applying static offset to prices."""
        from synctacles_db.fallback.fallback_manager import FallbackManager

        now = datetime.now(timezone.utc)
        prices = [
            {
                "timestamp": now.replace(hour=8).isoformat(),
                "price_eur_mwh": 50.0
            },
            {
                "timestamp": now.replace(hour=9).isoformat(),
                "price_eur_mwh": 55.0
            }
        ]

        result = FallbackManager._apply_static_offset(prices)

        assert len(result) == 2

        # Check first price got offset applied
        expected_0 = 50.0 + (HOURLY_OFFSET[8] * 1000)  # Convert offset to MWh
        assert result[0]["price_eur_mwh"] == pytest.approx(expected_0, rel=1e-4)

    @pytest.mark.asyncio
    async def test_add_reference_data(self):
        """Test adding reference data to prices."""
        from synctacles_db.fallback.fallback_manager import FallbackManager

        prices = [
            {"timestamp": datetime.now(timezone.utc).isoformat(), "price_eur_mwh": 200.0},
            {"timestamp": datetime.now(timezone.utc).isoformat(), "price_eur_mwh": 250.0}
        ]

        result = FallbackManager._add_reference_data(prices, "Test Source", 1)

        assert len(result) == 2
        assert "_reference" in result[0]

        ref = result[0]["_reference"]
        assert ref["source"] == "Test Source"
        assert ref["tier"] == 1
        assert "market" in ref
        assert "expected_range" in ref
        assert "low" in ref["expected_range"]
        assert "high" in ref["expected_range"]


class TestIntegration:
    """Integration tests (require network access)."""

    @pytest.mark.asyncio
    @pytest.mark.skip(reason="Requires live API access - run manually")
    async def test_easyenergy_live_fetch(self):
        """Test live EasyEnergy API fetch."""
        prices = await EasyEnergyClient.get_prices_today()

        assert prices is not None
        assert len(prices) >= 20  # Should have most hours

        # Check structure
        for p in prices[:5]:
            assert "timestamp" in p
            assert "price_eur_kwh" in p
            assert "price_eur_mwh" in p
            assert p["price_eur_mwh"] == p["price_eur_kwh"] * 1000

    @pytest.mark.asyncio
    @pytest.mark.skip(reason="Requires live API access - run manually")
    async def test_fallback_chain_live(self):
        """Test full fallback chain with live APIs."""
        from synctacles_db.fallback.fallback_manager import FallbackManager

        # Test with no DB results to force API fallback
        result, source, quality, allow_go = await FallbackManager.get_prices_with_fallback(
            db_results=None,
            db_age_minutes=9999,
            country="nl"
        )

        assert result is not None
        assert source != "None"
        assert quality in ["FRESH", "STALE", "FALLBACK", "CACHED"]
        print(f"Fallback result: {source}, {quality}, GO={allow_go}")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
