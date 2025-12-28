"""
Quick FastAPI starter for testing
Run: python3 start_api.py
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
