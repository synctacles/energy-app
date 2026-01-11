"""Add price_cache table for 24h fallback persistence

Revision ID: 20260111_price_cache
Revises: 20260102_archive_tennet_byo_migration
Create Date: 2026-01-11

Issue: #61 - Add 24h price cache for fallback
"""
from alembic import op
import sqlalchemy as sa

revision = '20260111_price_cache'
down_revision = '20260102_archive_tennet_byo_migration'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'price_cache',
        sa.Column('id', sa.Integer(), primary_key=True, autoincrement=True),
        sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
        sa.Column('country', sa.String(2), nullable=False, server_default='NL'),
        sa.Column('price_eur_kwh', sa.Numeric(10, 6), nullable=False),
        sa.Column('source', sa.String(50), nullable=False),
        sa.Column('quality', sa.String(20), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
    )

    op.create_index('idx_price_cache_timestamp', 'price_cache', ['timestamp'], postgresql_using='btree')
    op.create_index('idx_price_cache_country_timestamp', 'price_cache', ['country', sa.text('timestamp DESC')])


def downgrade():
    op.drop_index('idx_price_cache_country_timestamp')
    op.drop_index('idx_price_cache_timestamp')
    op.drop_table('price_cache')
