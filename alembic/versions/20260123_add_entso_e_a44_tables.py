"""Add ENTSO-E A44 (day-ahead prices) tables

Revision ID: 20260123_a44_prices
Revises: 20260113_frank_prices
Create Date: 2026-01-23

Note: These tables were missing from the original migration chain.
A44 = Day-ahead prices from ENTSO-E Transparency Platform.
"""
from alembic import op
import sqlalchemy as sa

revision = '20260123_a44_prices'
down_revision = '20260113_frank_prices'
branch_labels = None
depends_on = None


def upgrade():
    # Raw ENTSO-E A44 prices (day-ahead)
    op.create_table(
        'raw_entso_e_a44',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
        sa.Column('country', sa.String(2), nullable=False),
        sa.Column('price_eur_mwh', sa.Numeric(10, 2), nullable=True),
        sa.Column('xml_file', sa.String(255), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )

    # Normalized ENTSO-E A44 prices
    op.create_table(
        'norm_entso_e_a44',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
        sa.Column('country', sa.String(2), nullable=False),
        sa.Column('price_eur_mwh', sa.Numeric(10, 2), nullable=True),
        sa.Column('data_source', sa.String(20), server_default='ENTSO-E', nullable=True),
        sa.Column('data_quality', sa.String(20), server_default='OK', nullable=True),
        sa.Column('needs_backfill', sa.Boolean(), server_default='false', nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )

    # Indexes for query optimization
    op.create_index('idx_norm_a44_timestamp', 'norm_entso_e_a44', ['timestamp'], postgresql_using='btree', postgresql_ops={'timestamp': 'DESC'})
    op.create_index('idx_prices_country_time', 'norm_entso_e_a44', ['country', 'timestamp'], postgresql_using='btree', postgresql_ops={'timestamp': 'DESC'})
    op.create_unique_constraint('uq_prices_time_country', 'norm_entso_e_a44', ['timestamp', 'country'])


def downgrade():
    op.drop_table('norm_entso_e_a44')
    op.drop_table('raw_entso_e_a44')
