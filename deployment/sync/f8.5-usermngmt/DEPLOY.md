# F8.5 User Management - Deployment Manifest

## Files to Deploy

### Core Auth
- `auth_models.py` → `/opt/synctacles/app/synctacles_db/`
- `auth_service.py` → `/opt/synctacles/app/synctacles_db/`

### API Layer
- `api/middleware.py` → `/opt/synctacles/app/synctacles_db/api/`
- `api/auth.py` → `/opt/synctacles/app/synctacles_db/api/endpoints/`

### Database
- `alembic/20251220_add_user_auth.py` → `/opt/synctacles/app/alembic/versions/`

### Scripts
- `scripts/cleanup_api_usage.sh` → `/opt/synctacles/scripts/`

## Post-Deploy Steps

1. Run migrations:
```bash
   cd /opt/synctacles/app
   source /opt/synctacles/venv/bin/activate
   alembic upgrade head
```

2. Add admin key to .env:
```bash
   echo "ADMIN_API_KEY=$(openssl rand -hex 32)" >> /opt/synctacles/.env
```

3. Update systemd service:
```bash
   # Add to synctacles-api.service:
   EnvironmentFile=/opt/synctacles/.env
   systemctl daemon-reload
   systemctl restart synctacles-api
```

4. Setup cron job:
```bash
   echo "30 3 * * * /opt/synctacles/scripts/cleanup_api_usage.sh >> /opt/synctacles/logs/cleanup.log 2>&1" | crontab -u synctacles -
```

## Dependencies

Add to requirements.txt:
- email-validator==2.3.0
- dnspython==2.8.0

## Validation
```bash
# Test signup
curl -X POST http://localhost:8000/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'

# Test admin endpoint
curl http://localhost:8000/auth/admin/users \
  -H "X-Admin-Key: $ADMIN_API_KEY"
```
