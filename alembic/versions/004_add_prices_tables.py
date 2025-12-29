"""add_prices_tables

Revision ID: xxx
Revises: 20251220_user_auth
Create Date: 2025-12-22
"""
from alembic import op
import sqlalchemy as sa

revision = 'xxx'
down_revision = '20251220_user_auth'

def upgrade():
    # Raw prices table - composite primary key with timestamp (required for TimescaleDB hypertables)
    op.create_table(
        'raw_prices',
        sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
        sa.Column('country', sa.String(2), nullable=False),
        sa.Column('source', sa.String(50), nullable=False),  # 'energy-charts' or 'entso-e'
        sa.Column('price_eur_mwh', sa.Numeric(10, 2), nullable=False),
        sa.Column('fetch_time', sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column('source_file', sa.String(255)),
        sa.PrimaryKeyConstraint('timestamp', 'country', 'source', name='pk_raw_prices')
    )

    # Normalized prices table - composite primary key with timestamp (required for TimescaleDB hypertables)
    op.create_table(
        'norm_prices',
        sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
        sa.Column('country', sa.String(2), nullable=False),
        sa.Column('price_eur_mwh', sa.Numeric(10, 2), nullable=False),
        sa.Column('quality_status', sa.String(20), server_default='OK'),
        sa.Column('normalized_at', sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.PrimaryKeyConstraint('timestamp', 'country', name='pk_norm_prices')
    )

    # Create hypertable for norm_prices (now works - timestamp is in primary key)
    op.execute("SELECT create_hypertable('norm_prices', 'timestamp', if_not_exists => TRUE);")

    # Indexes for query optimization (primary key already covers timestamp, country)
    op.create_index('idx_raw_prices_source_time', 'raw_prices', ['source', 'timestamp'])
    op.create_index('idx_norm_prices_country_time', 'norm_prices', ['country', 'timestamp'])

def downgrade():
    op.drop_table('norm_prices')
    op.drop_table('raw_prices')