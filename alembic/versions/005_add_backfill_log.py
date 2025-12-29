"""Add backfill_log table for audit trail

Revision ID: 005_add_backfill_log
Revises: 004_add_prices_tables
Create Date: 2025-12-29 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '005_add_backfill_log'
down_revision = '004_add_prices_tables'
branch_labels = None
depends_on = None


def upgrade():
    """Create backfill_log table"""
    op.create_table(
        'backfill_log',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('source_type', sa.String(50), nullable=False,
                  doc='Data source: a75, a65, a44, energy_charts'),
        sa.Column('data_category', sa.String(50), nullable=True,
                  doc='PSR type (B01-B20), platform (aFRR, etc), or data type'),
        sa.Column('country', sa.String(2), nullable=False, server_default='NL'),
        sa.Column('gap_start', sa.DateTime(timezone=True), nullable=False),
        sa.Column('gap_end', sa.DateTime(timezone=True), nullable=False),
        sa.Column('backfill_time', sa.DateTime(timezone=True),
                  server_default=sa.func.now(), nullable=False),
        sa.Column('status', sa.String(20), nullable=False,
                  doc='SUCCESS, PARTIAL, FAILED, NO_DATA'),
        sa.Column('records_inserted', sa.Integer(), nullable=True),
        sa.Column('records_failed', sa.Integer(), nullable=True),
        sa.Column('error_message', sa.String(500), nullable=True),
        sa.Column('execution_duration_seconds', sa.Float(), nullable=True),
        sa.Column('fallback_source_used', sa.String(50), nullable=True,
                  doc='If fallback was used, which source (e.g., energy_charts)'),
        sa.PrimaryKeyConstraint('id')
    )

    # Create indexes for efficient querying
    op.create_index('ix_backfill_log_source_type', 'backfill_log', ['source_type'])
    op.create_index('ix_backfill_log_backfill_time', 'backfill_log', ['backfill_time'])
    op.create_index('ix_backfill_log_gap_range', 'backfill_log', ['gap_start', 'gap_end'])
    op.create_index('ix_backfill_log_status', 'backfill_log', ['status'])


def downgrade():
    """Drop backfill_log table"""
    op.drop_index('ix_backfill_log_status', table_name='backfill_log')
    op.drop_index('ix_backfill_log_gap_range', table_name='backfill_log')
    op.drop_index('ix_backfill_log_backfill_time', table_name='backfill_log')
    op.drop_index('ix_backfill_log_source_type', table_name='backfill_log')
    op.drop_table('backfill_log')
