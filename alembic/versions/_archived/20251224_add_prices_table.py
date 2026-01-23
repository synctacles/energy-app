"""Add prices table (A44 day-ahead)

Revision ID: 20251224_prices
Revises: 003add_norm_uq
Create Date: 2025-12-24

"""
from alembic import op
import sqlalchemy as sa

revision = '20251224_prices'
down_revision = '003add_norm_uq'
branch_labels = None
depends_on = None


def upgrade():
    # Raw table
    op.create_table(
        'raw_entso_e_a44',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('timestamp', sa.TIMESTAMP(timezone=True), nullable=False),
        sa.Column('country', sa.VARCHAR(2), nullable=False),
        sa.Column('price_eur_mwh', sa.NUMERIC(10, 2), nullable=True),
        sa.Column('xml_file', sa.VARCHAR(255), nullable=True),
        sa.Column('created_at', sa.TIMESTAMP(timezone=True), server_default=sa.text('NOW()'), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )
    
    # Normalized table
    op.create_table(
        'norm_entso_e_a44',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('timestamp', sa.TIMESTAMP(timezone=True), nullable=False),
        sa.Column('country', sa.VARCHAR(2), nullable=False),
        sa.Column('price_eur_mwh', sa.NUMERIC(10, 2), nullable=True),
        sa.Column('data_source', sa.VARCHAR(20), server_default='ENTSO-E', nullable=True),
        sa.Column('data_quality', sa.VARCHAR(20), server_default='OK', nullable=True),
        sa.Column('needs_backfill', sa.Boolean(), server_default='false', nullable=True),
        sa.Column('created_at', sa.TIMESTAMP(timezone=True), server_default=sa.text('NOW()'), nullable=False),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('timestamp', 'country', name='uq_prices_time_country')
    )
    
    # Index for queries
    op.create_index('idx_prices_country_time', 'norm_entso_e_a44', ['country', sa.text('timestamp DESC')])


def downgrade():
    op.drop_index('idx_prices_country_time', table_name='norm_entso_e_a44')
    op.drop_table('norm_entso_e_a44')
    op.drop_table('raw_entso_e_a44')
