#!/usr/bin/env python3
"""
FastAPI Application Starter (Wrapper)

This is a convenience wrapper that forwards to the actual implementation in scripts/start_api.py
to maintain backward compatibility after repository reorganization.

Usage:
    python3 start_api.py

The actual implementation can also be run directly:
    python3 scripts/start_api.py

For more information, see docs/README.md or the FastAPI documentation.
"""

import sys
from pathlib import Path

# Ensure the scripts directory is in the path for imports
scripts_dir = Path(__file__).parent / "scripts"
if str(scripts_dir) not in sys.path:
    sys.path.insert(0, str(scripts_dir))

# Import and execute the main start_api script
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "synctacles_db.api.main:app", host="0.0.0.0", port=8000, workers=4, reload=False
    )
