"""Archive TenneT tables for BYO-key migration

Revision ID: 20260102_tennet_archive
Revises: 005_add_backfill_log
Create Date: 2026-01-02 12:00:00.000000

This migration archives TenneT tables as part of the BYO-key (Bring Your Own) migration.
TenneT API license prohibits server-side redistribution, so balance data is now
fetched locally via Home Assistant component with user's personal API key.

Actions:
- Rename raw_tennet_balance → archive_raw_tennet_balance
- Rename norm_tennet_balance → archive_norm_tennet_balance

These tables are preserved for historical reference but no longer actively used.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '20260102_tennet_archive'
down_revision = '005_add_backfill_log'
branch_labels = None
depends_on = None


def upgrade():
    """
    Archive TenneT tables
    """
    # Rename raw_tennet_balance to archive_raw_tennet_balance
    op.execute("""
        ALTER TABLE IF EXISTS raw_tennet_balance
        RENAME TO archive_raw_tennet_balance
    """)

    # Rename norm_tennet_balance to archive_norm_tennet_balance
    op.execute("""
        ALTER TABLE IF EXISTS norm_tennet_balance
        RENAME TO archive_norm_tennet_balance
    """)

    # Add comment to tables for documentation
    op.execute("""
        COMMENT ON TABLE archive_raw_tennet_balance IS
        'ARCHIVED: TenneT balance delta data. No longer actively used.
         Migration: TenneT moved to BYO-key model (Home Assistant component).
         Preserved for historical reference.'
    """)

    op.execute("""
        COMMENT ON TABLE archive_norm_tennet_balance IS
        'ARCHIVED: Normalized TenneT balance data. No longer actively used.
         Migration: TenneT moved to BYO-key model (Home Assistant component).
         Preserved for historical reference.'
    """)


def downgrade():
    """
    Restore TenneT tables (not recommended in production)
    """
    # Rename archive_raw_tennet_balance back to raw_tennet_balance
    op.execute("""
        ALTER TABLE IF EXISTS archive_raw_tennet_balance
        RENAME TO raw_tennet_balance
    """)

    # Rename archive_norm_tennet_balance back to norm_tennet_balance
    op.execute("""
        ALTER TABLE IF EXISTS archive_norm_tennet_balance
        RENAME TO norm_tennet_balance
    """)
