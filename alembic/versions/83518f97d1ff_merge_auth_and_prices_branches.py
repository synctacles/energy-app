"""merge auth and prices branches

Revision ID: 83518f97d1ff
Revises: 20251224_prices, xxx
Create Date: 2025-12-24 02:00:55.965418

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '83518f97d1ff'
down_revision: Union[str, Sequence[str], None] = ('20251224_prices', 'xxx')
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    """Upgrade schema."""
    pass


def downgrade() -> None:
    """Downgrade schema."""
    pass
