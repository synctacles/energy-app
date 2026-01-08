"""
Pipeline health endpoint for Grafana monitoring.
KISS approach: JSON endpoint, no Prometheus complexity.
"""
from fastapi import APIRouter, Depends
from fastapi.responses import Response
from datetime import datetime, timezone
import subprocess
from sqlalchemy import text
from sqlalchemy.orm import Session
from synctacles_db.api.dependencies import get_db
from prometheus_client import Gauge, generate_latest, CONTENT_TYPE_LATEST, CollectorRegistry

router = APIRouter(prefix="/v1/pipeline", tags=["pipeline"])

# Dedicated registry for pipeline metrics (avoid conflicts with main app metrics)
pipeline_registry = CollectorRegistry()

# Pipeline health metrics
timer_status_gauge = Gauge(
    'pipeline_timer_status',
    'Timer status (1=active, 0=stopped)',
    ['timer'],
    registry=pipeline_registry
)

timer_last_trigger_minutes = Gauge(
    'pipeline_timer_last_trigger_minutes',
    'Minutes since timer last triggered',
    ['timer'],
    registry=pipeline_registry
)

data_freshness_minutes = Gauge(
    'pipeline_data_freshness_minutes',
    'Data age in minutes (normalized table)',
    ['source'],
    registry=pipeline_registry
)

data_status_gauge = Gauge(
    'pipeline_data_status',
    'Data status (0=FRESH, 1=STALE, 2=UNAVAILABLE, 3=NO_DATA)',
    ['source'],
    registry=pipeline_registry
)


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
    """Get data freshness from database (historical data only, excludes forecasts)."""
    # Raw data age - only historical (timestamp <= NOW)
    raw_result = session.execute(text(f"""
        SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
        FROM {raw_table}
        WHERE timestamp <= NOW()
    """)).fetchone()
    raw_age = round(raw_result[0], 1) if raw_result and raw_result[0] else None

    # Normalized data age - only historical (timestamp <= NOW)
    norm_result = session.execute(text(f"""
        SELECT EXTRACT(EPOCH FROM (NOW() - MAX(timestamp)))/60 as age_min
        FROM {norm_table}
        WHERE timestamp <= NOW()
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


@router.get("/metrics")
def pipeline_metrics(db: Session = Depends(get_db)):
    """
    Prometheus metrics endpoint for pipeline health.

    Exposes timer status and data freshness as Prometheus gauges.
    """
    # Get timer statuses
    timers = {
        "collector": get_timer_status("collector"),
        "importer": get_timer_status("importer"),
        "normalizer": get_timer_status("normalizer"),
        "health": get_timer_status("health")
    }

    # Update timer metrics
    for timer_name, status in timers.items():
        timer_status_gauge.labels(timer=timer_name).set(1 if status["active"] else 0)
        if status["last_trigger_ago_min"] is not None:
            timer_last_trigger_minutes.labels(timer=timer_name).set(status["last_trigger_ago_min"])

    # Get data freshness
    data_sources = {
        "a75": get_data_freshness(db, "a75", "raw_entso_e_a75", "norm_entso_e_a75"),
        "a65": get_data_freshness(db, "a65", "raw_entso_e_a65", "norm_entso_e_a65"),
        "a44": get_data_freshness(db, "a44", "raw_entso_e_a44", "norm_entso_e_a44")
    }

    # Update data metrics
    status_map = {"FRESH": 0, "STALE": 1, "UNAVAILABLE": 2, "NO_DATA": 3}
    for source, data in data_sources.items():
        if data["norm_age_min"] is not None:
            data_freshness_minutes.labels(source=source).set(data["norm_age_min"])
        data_status_gauge.labels(source=source).set(status_map.get(data["status"], 3))

    # Generate Prometheus format output
    return Response(
        content=generate_latest(pipeline_registry),
        media_type=CONTENT_TYPE_LATEST
    )
