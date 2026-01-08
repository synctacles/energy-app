"""
Pipeline health endpoint for Grafana monitoring.
KISS approach: JSON endpoint, no Prometheus complexity.
"""
from fastapi import APIRouter, Depends
from datetime import datetime, timezone
import subprocess
from sqlalchemy import text
from sqlalchemy.orm import Session
from synctacles_db.api.dependencies import get_db

router = APIRouter(prefix="/v1/pipeline", tags=["pipeline"])


def get_timer_status(timer_name: str) -> dict:
    """Get systemd timer status."""
    full_name = f"energy-insights-nl-{timer_name}.timer"

    # Check if active
    result = subprocess.run(
        ["systemctl", "is-active", full_name],
        capture_output=True, text=True
    )
    is_active = result.stdout.strip() == "active"

    # Get last trigger time
    result = subprocess.run(
        ["systemctl", "show", full_name, "--property=LastTriggerUSec"],
        capture_output=True, text=True
    )
    last_trigger = None
    last_trigger_ago_min = None

    if "LastTriggerUSec=" in result.stdout:
        timestamp_str = result.stdout.strip().split("=")[1]
        if timestamp_str and timestamp_str != "n/a":
            try:
                # Parse systemd timestamp
                last_trigger = timestamp_str
                # Calculate minutes ago (simplified)
                result2 = subprocess.run(
                    ["systemctl", "show", full_name, "--property=LastTriggerUSecMonotonic"],
                    capture_output=True, text=True
                )
                if "=" in result2.stdout:
                    mono_usec = int(result2.stdout.strip().split("=")[1])
                    # Get current monotonic time
                    with open("/proc/uptime") as f:
                        uptime_sec = float(f.read().split()[0])
                    current_mono_usec = int(uptime_sec * 1_000_000)
                    age_min = (current_mono_usec - mono_usec) / 60_000_000
                    last_trigger_ago_min = round(age_min, 1)
            except:
                pass

    return {
        "active": is_active,
        "last_trigger": last_trigger,
        "last_trigger_ago_min": last_trigger_ago_min,
        "status": "OK" if is_active else "STOPPED"
    }


def get_data_freshness(session: Session, source: str, raw_table: str, norm_table: str) -> dict:
    """Get data freshness from database."""
    # Raw data age (created_at)
    raw_result = session.execute(text(f"""
        SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
        FROM {raw_table}
    """)).fetchone()
    raw_age = round(raw_result[0], 1) if raw_result and raw_result[0] else None

    # Normalized data age (timestamp)
    norm_result = session.execute(text(f"""
        SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
        FROM {norm_table}
    """)).fetchone()
    norm_age = round(norm_result[0], 1) if norm_result and norm_result[0] else None

    # Determine status
    if norm_age is None:
        status = "NO_DATA"
    elif norm_age < 90:
        status = "FRESH"
    elif norm_age < 180:
        status = "STALE"
    else:
        status = "UNAVAILABLE"

    # Detect pipeline gap (raw OK but norm stale = normalizer issue)
    pipeline_gap = None
    if raw_age is not None and norm_age is not None:
        pipeline_gap = round(norm_age - raw_age, 1)

    return {
        "raw_age_min": raw_age,
        "norm_age_min": norm_age,
        "pipeline_gap_min": pipeline_gap,
        "status": status
    }


@router.get("/health")
def pipeline_health(db: Session = Depends(get_db)):
    """
    Complete pipeline health status for Grafana dashboard.

    Returns timer status and data freshness for all sources.
    """
    now = datetime.now(timezone.utc).isoformat()

    return {
        "timestamp": now,
        "timers": {
            "collector": get_timer_status("collector"),
            "importer": get_timer_status("importer"),
            "normalizer": get_timer_status("normalizer"),
            "health": get_timer_status("health")
        },
        "data": {
            "a75": get_data_freshness(db, "a75", "raw_entso_e_a75", "norm_entso_e_a75"),
            "a65": get_data_freshness(db, "a65", "raw_entso_e_a65", "norm_entso_e_a65"),
            "a44": get_data_freshness(db, "a44", "raw_entso_e_a44", "norm_entso_e_a44")
        },
        "api": {
            "status": "OK",
            "workers": 8
        }
    }
