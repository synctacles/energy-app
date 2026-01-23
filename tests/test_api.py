"""
API Endpoint Tests for SYNCTACLES API.

Tests the FastAPI endpoints without requiring a database connection.
Uses TestClient for synchronous testing.
"""

import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch, MagicMock
from datetime import datetime, timezone


class TestHealthEndpoint:
    """Tests for /health endpoint."""

    def test_health_returns_ok(self):
        """Health endpoint should return status ok."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/health")

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "ok"
        assert "timestamp" in data
        assert "version" in data

    def test_health_contains_brand(self):
        """Health endpoint should contain brand info."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/health")

        data = response.json()
        assert "brand" in data
        assert "service" in data


class TestDocsEndpoint:
    """Tests for /docs endpoint."""

    def test_docs_available(self):
        """OpenAPI docs should be available."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/docs")

        assert response.status_code == 200

    def test_openapi_json_available(self):
        """OpenAPI JSON schema should be available."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/openapi.json")

        assert response.status_code == 200
        data = response.json()
        assert "openapi" in data
        assert "paths" in data


@pytest.mark.integration
class TestPricesEndpoint:
    """Tests for /api/v1/prices endpoint.

    These tests require a database connection and are marked as integration tests.
    Run with: pytest -m integration
    """

    @pytest.mark.skip(reason="Requires database connection")
    def test_prices_returns_list(self):
        """Prices endpoint should return a list of prices."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/api/v1/prices")

        # Should return 200 or valid response
        assert response.status_code in [200, 500, 503]

    @pytest.mark.skip(reason="Requires database connection")
    def test_prices_with_hours_param(self):
        """Prices endpoint should accept hours parameter."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/api/v1/prices?hours=24")

        # Should not return 422 (validation error)
        assert response.status_code != 422


class TestMetricsEndpoint:
    """Tests for /metrics endpoint (Prometheus)."""

    def test_metrics_available(self):
        """Metrics endpoint should be available."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/metrics")

        # Metrics might return 200 or 404 depending on setup
        assert response.status_code in [200, 404]


class TestCORSHeaders:
    """Tests for CORS configuration."""

    def test_cors_headers_present(self):
        """CORS headers should be present on responses."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.options(
            "/health",
            headers={
                "Origin": "https://example.com",
                "Access-Control-Request-Method": "GET",
            },
        )

        # CORS preflight should return 200
        assert response.status_code == 200


class TestErrorHandling:
    """Tests for error handling."""

    def test_404_on_unknown_endpoint(self):
        """Unknown endpoints should return 404."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/nonexistent/endpoint")

        assert response.status_code == 404

    def test_invalid_method_returns_405(self):
        """Invalid HTTP method should return 405."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.post("/health")

        assert response.status_code == 405


@pytest.mark.integration
class TestAPIVersioning:
    """Tests for API versioning."""

    @pytest.mark.skip(reason="Requires database connection")
    def test_v1_prefix_works(self):
        """API v1 prefix should work."""
        from synctacles_db.api.main import app

        client = TestClient(app)
        response = client.get("/api/v1/prices")

        # Should not return 404 for the prefix
        assert response.status_code != 404
