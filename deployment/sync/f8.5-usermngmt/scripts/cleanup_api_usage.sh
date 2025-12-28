#!/bin/bash
cd /opt/synctacles/app
source /opt/synctacles/venv/bin/activate
python3 << 'PYTHON'
from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db
db = next(get_db())
deleted = auth_service.cleanup_old_usage_logs(db, days=30)
print(f"✓ Cleaned {deleted} old usage records")
PYTHON
