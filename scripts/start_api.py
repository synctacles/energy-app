"""
FastAPI Application Implementation

This is the actual FastAPI startup script. It can be run directly or via the wrapper:

Direct (from scripts/):
    python3 start_api.py

Via wrapper (from root):
    python3 start_api.py

For production deployments, consider using:
    gunicorn synctacles_db.api.main:app --workers 4 --bind 0.0.0.0:8000
"""
import uvicorn

if __name__ == "__main__":
    uvicorn.run(
        "synctacles_db.api.main:app",
        host="0.0.0.0",
        port=8000,
        workers=4,
        reload=False
    )
