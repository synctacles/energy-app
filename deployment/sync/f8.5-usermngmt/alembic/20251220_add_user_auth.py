"""add user authentication tables

Revision ID: 20251220_user_auth
Revises: 003add_norm_uq
Create Date: 2025-12-21

"""
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import UUID

from alembic import op

revision = '20251220_user_auth'
down_revision = '003add_norm_uq'
branch_labels = None
depends_on = None


def upgrade():
    # Users table
    op.create_table(
        'users',
        sa.Column('id', UUID(as_uuid=True), primary_key=True, server_default=sa.text('gen_random_uuid()')),
        sa.Column('email', sa.String(255), unique=True, nullable=False),
        sa.Column('license_key', UUID(as_uuid=True), unique=True, nullable=False, server_default=sa.text('gen_random_uuid()')),
        sa.Column('api_key_hash', sa.String(64), unique=True, nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column('is_active', sa.Boolean, default=True, server_default='true'),
        sa.Column('tier', sa.String(20), default='free', server_default='free'),
        sa.Column('rate_limit_daily', sa.Integer, default=1000, server_default='1000')
    )

    # API usage tracking
    op.create_table(
        'api_usage',
        sa.Column('id', sa.Integer, primary_key=True, autoincrement=True),
        sa.Column('user_id', UUID(as_uuid=True), sa.ForeignKey('users.id', ondelete='CASCADE')),
        sa.Column('endpoint', sa.String(255), nullable=False),
        sa.Column('timestamp', sa.DateTime(timezone=True), server_default=sa.func.now()),
        sa.Column('status_code', sa.Integer),
    )

    # Indexes
    op.create_index('idx_users_email', 'users', ['email'])
    op.create_index('idx_users_api_key', 'users', ['api_key_hash'])
    op.create_index('idx_usage_user_timestamp', 'api_usage', ['user_id', 'timestamp'])


def downgrade():
    op.drop_table('api_usage')
    op.drop_table('users')
