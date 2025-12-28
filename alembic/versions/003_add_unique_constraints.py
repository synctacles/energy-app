"""003_add_norm_unique_constraints

Revision ID: 003_add_norm_unique_constraints
Revises: 002_add_unique_constraints
Create Date: 2025-12-17 11:30:00.000000
"""
from typing import Sequence, Union
from alembic import op
import sqlalchemy as sa

revision: str = '003add_norm_uq'
down_revision: Union[str, None] = '002_add_unique_constraints'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None

def upgrade() -> None:
    """Add country column to TenneT + UNIQUE constraints for all norm_* tables."""
    
    # STEP 1: Add country column to norm_tennet_balance (if missing)
    # Check if column exists to make migration idempotent
    from sqlalchemy import inspect
    from alembic import op
    
    conn = op.get_bind()
    inspector = inspect(conn)
    columns = [col['name'] for col in inspector.get_columns('norm_tennet_balance')]
    
    if 'country' not in columns:
        op.add_column('norm_tennet_balance', 
            sa.Column('country', sa.String(2), nullable=False, server_default='NL')
        )
    
    # STEP 2: Add UNIQUE constraints
    op.create_unique_constraint(
        'uq_norm_entso_e_a75_natural_key',
        'norm_entso_e_a75',
        ['timestamp', 'country']
    )
    
    op.create_unique_constraint(
        'uq_norm_entso_e_a65_natural_key',
        'norm_entso_e_a65',
        ['timestamp', 'country']
    )
    
    op.create_unique_constraint(
        'uq_norm_tennet_balance_natural_key',
        'norm_tennet_balance',
        ['timestamp', 'country']
    )

def downgrade() -> None:
    """Remove UNIQUE constraints + country column from TenneT."""
    
    # Remove UNIQUE constraints (reverse order of creation)
    op.drop_constraint('uq_norm_tennet_balance_natural_key', 'norm_tennet_balance', type_='unique')
    op.drop_constraint('uq_norm_entso_e_a65_natural_key', 'norm_entso_e_a65', type_='unique')
    op.drop_constraint('uq_norm_entso_e_a75_natural_key', 'norm_entso_e_a75', type_='unique')
    
    # Remove country column from norm_tennet_balance
    op.drop_column('norm_tennet_balance', 'country')
