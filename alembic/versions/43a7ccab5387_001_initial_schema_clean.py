"""001_initial_schema_clean

Revision ID: 43a7ccab5387
Revises:
Create Date: 2025-12-10 17:41:27.695168

Updated: 2026-01-23 - Removed TenneT tables (discontinued)
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '43a7ccab5387'
down_revision: Union[str, Sequence[str], None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Upgrade schema."""
    # fetch_log - tracks data collection operations
    op.create_table('fetch_log',
    sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
    sa.Column('source', sa.String(length=50), nullable=False),
    sa.Column('fetch_time', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=False),
    sa.Column('status', sa.String(length=20), nullable=False),
    sa.Column('records_fetched', sa.Integer(), nullable=True),
    sa.Column('error_message', sa.String(length=500), nullable=True),
    sa.PrimaryKeyConstraint('id')
    )
    op.create_index('ix_fetch_log_fetch_time', 'fetch_log', ['fetch_time'], unique=False)
    op.create_index('ix_fetch_log_source', 'fetch_log', ['source'], unique=False)

    # norm_entso_e_a65 - normalized load data
    op.create_table('norm_entso_e_a65',
    sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
    sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
    sa.Column('country', sa.String(length=2), nullable=False),
    sa.Column('actual_mw', sa.Float(), nullable=True),
    sa.Column('forecast_mw', sa.Float(), nullable=True),
    sa.Column('quality_status', sa.String(length=20), nullable=True),
    sa.Column('last_updated', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=True),
    sa.PrimaryKeyConstraint('id', 'timestamp')
    )

    # norm_entso_e_a75 - normalized generation mix
    op.create_table('norm_entso_e_a75',
    sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
    sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
    sa.Column('country', sa.String(length=2), nullable=False),
    sa.Column('b01_biomass_mw', sa.Float(), nullable=True),
    sa.Column('b04_gas_mw', sa.Float(), nullable=True),
    sa.Column('b05_coal_mw', sa.Float(), nullable=True),
    sa.Column('b14_nuclear_mw', sa.Float(), nullable=True),
    sa.Column('b16_solar_mw', sa.Float(), nullable=True),
    sa.Column('b17_waste_mw', sa.Float(), nullable=True),
    sa.Column('b18_wind_offshore_mw', sa.Float(), nullable=True),
    sa.Column('b19_wind_onshore_mw', sa.Float(), nullable=True),
    sa.Column('b20_other_mw', sa.Float(), nullable=True),
    sa.Column('total_mw', sa.Float(), nullable=True),
    sa.Column('quality_status', sa.String(length=20), nullable=True),
    sa.Column('last_updated', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=True),
    sa.PrimaryKeyConstraint('id', 'timestamp')
    )

    # raw_entso_e_a65 - raw load data
    op.create_table('raw_entso_e_a65',
    sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
    sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
    sa.Column('country', sa.String(length=2), nullable=False),
    sa.Column('type', sa.String(length=20), nullable=False),
    sa.Column('quantity_mw', sa.Float(), nullable=False),
    sa.Column('source_file', sa.String(length=255), nullable=True),
    sa.Column('imported_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=True),
    sa.PrimaryKeyConstraint('id')
    )
    op.create_index('ix_raw_entso_e_a65_timestamp', 'raw_entso_e_a65', ['timestamp'], unique=False)
    op.create_index('ix_raw_entso_e_a65_type', 'raw_entso_e_a65', ['type'], unique=False)

    # raw_entso_e_a75 - raw generation mix
    op.create_table('raw_entso_e_a75',
    sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
    sa.Column('timestamp', sa.DateTime(timezone=True), nullable=False),
    sa.Column('country', sa.String(length=2), nullable=False),
    sa.Column('psr_type', sa.String(length=3), nullable=False),
    sa.Column('quantity_mw', sa.Float(), nullable=False),
    sa.Column('source_file', sa.String(length=255), nullable=True),
    sa.Column('imported_at', sa.DateTime(timezone=True), server_default=sa.text('now()'), nullable=True),
    sa.PrimaryKeyConstraint('id')
    )
    op.create_index('ix_raw_entso_e_a75_psr_type', 'raw_entso_e_a75', ['psr_type'], unique=False)
    op.create_index('ix_raw_entso_e_a75_timestamp', 'raw_entso_e_a75', ['timestamp'], unique=False)


def downgrade() -> None:
    """Downgrade schema."""
    op.drop_index('ix_raw_entso_e_a75_timestamp', table_name='raw_entso_e_a75')
    op.drop_index('ix_raw_entso_e_a75_psr_type', table_name='raw_entso_e_a75')
    op.drop_table('raw_entso_e_a75')
    op.drop_index('ix_raw_entso_e_a65_type', table_name='raw_entso_e_a65')
    op.drop_index('ix_raw_entso_e_a65_timestamp', table_name='raw_entso_e_a65')
    op.drop_table('raw_entso_e_a65')
    op.drop_table('norm_entso_e_a75')
    op.drop_table('norm_entso_e_a65')
    op.drop_index('ix_fetch_log_source', table_name='fetch_log')
    op.drop_index('ix_fetch_log_fetch_time', table_name='fetch_log')
    op.drop_table('fetch_log')
