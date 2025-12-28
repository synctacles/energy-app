"""002_add_unique_constraints

Revision ID: 002_add_unique_constraints
Revises: 43a7ccab5387
Create Date: 2025-12-16 19:42:00.000000

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '002_add_unique_constraints'
down_revision: Union[str, Sequence[str], None] = '43a7ccab5387'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Add UNIQUE constraints for ON CONFLICT clauses in importers."""
    # Add UNIQUE constraint for raw_entso_e_a75: (timestamp, country, psr_type)
    op.create_unique_constraint(
        'uq_raw_entso_e_a75_natural_key',
        'raw_entso_e_a75',
        ['timestamp', 'country', 'psr_type']
    )
    
    # Add UNIQUE constraint for raw_entso_e_a65: (timestamp, country, type)
    op.create_unique_constraint(
        'uq_raw_entso_e_a65_natural_key',
        'raw_entso_e_a65',
        ['timestamp', 'country', 'type']
    )
    
    # Add UNIQUE constraint for raw_tennet_balance: (timestamp, platform)
    op.create_unique_constraint(
        'uq_raw_tennet_balance_natural_key',
        'raw_tennet_balance',
        ['timestamp', 'platform']
    )


def downgrade() -> None:
    """Remove UNIQUE constraints."""
    # Drop UNIQUE constraints in reverse order
    op.drop_constraint('uq_raw_tennet_balance_natural_key', 'raw_tennet_balance', type_='unique')
    op.drop_constraint('uq_raw_entso_e_a65_natural_key', 'raw_entso_e_a65', type_='unique')
    op.drop_constraint('uq_raw_entso_e_a75_natural_key', 'raw_entso_e_a75', type_='unique')
