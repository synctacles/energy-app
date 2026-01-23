"""005_add_backfill_log - DISCONTINUED

Revision ID: 005_add_backfill_log
Revises: 004_add_prices_tables
Create Date: 2025-12-29 00:00:00.000000

Note: 2026-01-23 - Backfill functionality discontinued.
      Migration kept as no-op to preserve revision chain.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '005_add_backfill_log'
down_revision = '004_add_prices_tables'
branch_labels = None
depends_on = None


def upgrade():
    """No-op: backfill functionality discontinued"""
    pass


def downgrade():
    """No-op: backfill functionality discontinued"""
    pass
