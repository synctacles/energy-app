"""Add frank_prices and enever_frank_prices tables for database-backed fallback

Revision ID: 20260113_frank_prices
Revises: 20260111_price_cache
Create Date: 2026-01-13

Purpose: Replace real-time API calls with database-backed fallback chain
- frank_prices: Direct Frank Energie consumer prices (Tier 1)
- enever_frank_prices: Enever-Frank prices via Coefficient server (Tier 2)
"""
from alembic import op
import sqlalchemy as sa

revision = '20260113_frank_prices'
down_revision = '20260111_price_cache'
branch_labels = None
depends_on = None


def upgrade():
    # Table 1: Frank Direct prices (Tier 1)
    op.create_table(
        'frank_prices',
        sa.Column('timestamp', sa.DateTime(timezone=True), primary_key=True),
        sa.Column('price_eur_kwh', sa.Numeric(10, 6), nullable=False),
        sa.Column('market_price', sa.Numeric(10, 6), nullable=True),
        sa.Column('market_price_tax', sa.Numeric(10, 6), nullable=True),
        sa.Column('sourcing_markup', sa.Numeric(10, 6), nullable=True),
        sa.Column('energy_tax', sa.Numeric(10, 6), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
    )

    op.create_index(
        'idx_frank_prices_timestamp',
        'frank_prices',
        [sa.text('timestamp DESC')],
        postgresql_using='btree'
    )

    # Table 2: Enever-Frank prices via Coefficient (Tier 2)
    op.create_table(
        'enever_frank_prices',
        sa.Column('timestamp', sa.DateTime(timezone=True), primary_key=True),
        sa.Column('price_eur_kwh', sa.Numeric(10, 6), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
    )

    op.create_index(
        'idx_enever_frank_prices_timestamp',
        'enever_frank_prices',
        [sa.text('timestamp DESC')],
        postgresql_using='btree'
    )


def downgrade():
    op.drop_index('idx_enever_frank_prices_timestamp')
    op.drop_table('enever_frank_prices')
    op.drop_index('idx_frank_prices_timestamp')
    op.drop_table('frank_prices')
