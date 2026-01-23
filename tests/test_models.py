"""
Model Tests for SYNCTACLES.

Tests the SQLAlchemy models without requiring a database connection.
Focuses on model structure, defaults, and relationships.
"""


class TestModelImports:
    """Test that all models can be imported."""

    def test_import_raw_models(self):
        """Raw models should be importable."""
        from synctacles_db.models import RawEntsoeA44

        assert RawEntsoeA44 is not None

    def test_import_norm_models(self):
        """Normalized models should be importable."""
        from synctacles_db.models import NormEntsoeA44

        assert NormEntsoeA44 is not None

    def test_import_price_models(self):
        """Price models should be importable."""
        from synctacles_db.models import (
            FrankPrices,
            PriceCache,
        )

        assert FrankPrices is not None
        assert PriceCache is not None

    def test_import_logging_models(self):
        """Logging models should be importable."""
        from synctacles_db.models import FetchLog

        assert FetchLog is not None


class TestRawEntsoeA44Model:
    """Tests for RawEntsoeA44 model."""

    def test_model_has_required_columns(self):
        """Model should have all required columns."""
        from synctacles_db.models import RawEntsoeA44

        columns = [c.name for c in RawEntsoeA44.__table__.columns]

        assert "id" in columns
        assert "timestamp" in columns
        assert "country" in columns
        assert "price_eur_mwh" in columns

    def test_model_tablename(self):
        """Model should have correct table name."""
        from synctacles_db.models import RawEntsoeA44

        assert RawEntsoeA44.__tablename__ == "raw_entso_e_a44"


class TestNormEntsoeA44Model:
    """Tests for NormEntsoeA44 model."""

    def test_model_has_required_columns(self):
        """Model should have required columns."""
        from synctacles_db.models import NormEntsoeA44

        columns = [c.name for c in NormEntsoeA44.__table__.columns]

        assert "id" in columns
        assert "timestamp" in columns
        assert "country" in columns
        assert "price_eur_mwh" in columns
        assert "data_source" in columns
        assert "data_quality" in columns

    def test_model_tablename(self):
        """Model should have correct table name."""
        from synctacles_db.models import NormEntsoeA44

        assert NormEntsoeA44.__tablename__ == "norm_entso_e_a44"


class TestFrankPricesModel:
    """Tests for FrankPrices model."""

    def test_model_has_price_columns(self):
        """Model should have price columns."""
        from synctacles_db.models import FrankPrices

        columns = [c.name for c in FrankPrices.__table__.columns]

        assert "timestamp" in columns
        assert "price_eur_kwh" in columns
        assert "market_price" in columns

    def test_model_tablename(self):
        """Model should have correct table name."""
        from synctacles_db.models import FrankPrices

        assert FrankPrices.__tablename__ == "frank_prices"


class TestPriceCacheModel:
    """Tests for PriceCache model."""

    def test_model_has_cache_columns(self):
        """Model should have cache-related columns."""
        from synctacles_db.models import PriceCache

        columns = [c.name for c in PriceCache.__table__.columns]

        assert "id" in columns
        assert "timestamp" in columns
        assert "price_eur_kwh" in columns
        assert "source" in columns
        assert "quality" in columns

    def test_model_tablename(self):
        """Model should have correct table name."""
        from synctacles_db.models import PriceCache

        assert PriceCache.__tablename__ == "price_cache"


class TestFetchLogModel:
    """Tests for FetchLog model."""

    def test_model_has_logging_columns(self):
        """Model should have logging columns."""
        from synctacles_db.models import FetchLog

        columns = [c.name for c in FetchLog.__table__.columns]

        assert "id" in columns
        assert "source" in columns
        assert "status" in columns
        assert "records_fetched" in columns

    def test_model_tablename(self):
        """Model should have correct table name."""
        from synctacles_db.models import FetchLog

        assert FetchLog.__tablename__ == "fetch_log"


class TestModelPrimaryKeys:
    """Tests for model primary keys."""

    def test_all_models_have_primary_keys(self):
        """All models should have primary keys."""
        from synctacles_db.models import (
            FetchLog,
            FrankPrices,
            NormEntsoeA44,
            PriceCache,
            RawEntsoeA44,
        )

        models = [RawEntsoeA44, NormEntsoeA44, FrankPrices, PriceCache, FetchLog]

        for model in models:
            pk_columns = [c for c in model.__table__.columns if c.primary_key]
            assert len(pk_columns) > 0, f"{model.__name__} has no primary key"
