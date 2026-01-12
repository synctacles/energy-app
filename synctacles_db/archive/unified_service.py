"""
Unified data service - combines generation, load, balance into single snapshot.
"""

from datetime import datetime, timezone
from typing import Optional
from sqlalchemy.orm import Session
from sqlalchemy import desc

from synctacles_db.models import NormEntsoeA75, NormEntsoeA65


def get_unified_snapshot(db: Session, country: str = "NL") -> dict:
    """
    Get unified data snapshot for most recent available timestamp.
    
    Args:
        db: Database session
        country: ISO country code (default: NL)
        
    Returns:
        dict with all components, structured per strict contract rules
    """
    aggregation_time = datetime.now(timezone.utc)
    
    # === GENERATION ===
    gen_record = db.query(NormEntsoeA75)\
        .filter(NormEntsoeA75.country == country)\
        .filter(NormEntsoeA75.timestamp <= aggregation_time)\
        .order_by(desc(NormEntsoeA75.timestamp))\
        .first()
    
    if gen_record:
        generation_total = sum([
            gen_record.b01_biomass_mw or 0,
            gen_record.b04_gas_mw or 0,
            gen_record.b05_coal_mw or 0,
            gen_record.b14_nuclear_mw or 0,
            gen_record.b16_solar_mw or 0,
            gen_record.b17_waste_mw or 0,
            gen_record.b18_wind_offshore_mw or 0,
            gen_record.b19_wind_onshore_mw or 0,
            gen_record.b20_other_mw or 0
        ])
        
        # STRICT renewables (exclude waste)
        renewable_strict = sum([
            gen_record.b01_biomass_mw or 0,
            gen_record.b16_solar_mw or 0,
            gen_record.b18_wind_offshore_mw or 0,
            gen_record.b19_wind_onshore_mw or 0
        ])
        
        renewable_pct = (renewable_strict / generation_total * 100) if generation_total > 0 else 0
        
        # OPTIONAL: incl waste
        renewable_incl_waste = renewable_strict + (gen_record.b17_waste_mw or 0)
        renewable_pct_incl_waste = (renewable_incl_waste / generation_total * 100) if generation_total > 0 else 0
        
        gen_freshness = (aggregation_time - gen_record.timestamp).total_seconds()
        gen_status = calculate_component_status(gen_freshness)
        gen_data = {
            "total_mw": round(generation_total, 2),
            "renewable_percentage": round(renewable_pct, 1),
            "renewable_percentage_incl_waste": round(renewable_pct_incl_waste, 1),
            "available": True,
            "status": gen_status,
            "freshness_seconds": int(gen_freshness),
            "timestamp": gen_record.timestamp.isoformat(),
            "reason": None
        }
    else:
        gen_status = "MISSING"
        gen_data = {
            "total_mw": None,
            "renewable_percentage": None,
            "renewable_percentage_incl_waste": None,
            "available": False,
            "status": gen_status,
            "freshness_seconds": None,
            "timestamp": None,
            "reason": "no_data"
        }
    
    # === LOAD ===
    load_record = db.query(NormEntsoeA65)\
        .filter(NormEntsoeA65.country == country)\
        .filter(NormEntsoeA65.timestamp <= aggregation_time)\
        .filter(NormEntsoeA65.actual_mw.isnot(None))\
        .order_by(desc(NormEntsoeA65.timestamp))\
        .first()
    
    if load_record:
        load_freshness = (aggregation_time - load_record.timestamp).total_seconds()
        load_status = calculate_component_status(load_freshness)
        load_data = {
            "actual_mw": round(load_record.actual_mw, 2) if load_record.actual_mw else None,
            "forecast_mw": round(load_record.forecast_mw, 2) if load_record.forecast_mw else None,
            "available": True,
            "status": load_status,
            "freshness_seconds": int(load_freshness),
            "timestamp": load_record.timestamp.isoformat(),
            "reason": None
        }
    else:
        load_status = "MISSING"
        load_data = {
            "actual_mw": None,
            "forecast_mw": None,
            "available": False,
            "status": load_status,
            "freshness_seconds": None,
            "timestamp": None,
            "reason": "no_data"
        }
    
    # === BALANCE (DEPRECATED - TenneT BYO-key model) ===
    # TenneT balance data is no longer collected server-side (ADR-001)
    # Users must configure their own TenneT API key in Home Assistant integration
    balance_status = "DEPRECATED"
    balance_data = {
        "delta_mw": None,
        "price_eur_mwh": None,
        "available": False,
        "status": balance_status,
        "freshness_seconds": None,
        "timestamp": None,
        "reason": "deprecated_tennet_byo_key"
    }
    
    # === OVERALL STATUS ===
    component_statuses = [gen_status, load_status, balance_status]
    overall_status = determine_worst_status(component_statuses)
    
    # === MISSING FIELDS ===
    missing_fields = []
    if not gen_data["available"]:
        missing_fields.extend(["generation.total_mw", "generation.renewable_percentage"])
    if not load_data["available"]:
        missing_fields.extend(["load.actual_mw"])
    if not balance_data["available"]:
        missing_fields.extend(["balance.delta_mw"])
    
    return {
        "timestamp": aggregation_time.isoformat(),
        "country": country,
        "generation": gen_data,
        "load": load_data,
        "balance": balance_data,
        "overall_status": overall_status,
        "missing_fields": missing_fields,
        "policy": {
            "waste_counts_as_renewable": False
        }
    }


def calculate_component_status(freshness_seconds: float) -> str:
    """
    Determine component status based on data age.
    
    OK       < 15 min (900s)
    DEGRADED 15-60 min (3600s)
    STALE    > 60 min
    """
    if freshness_seconds < 900:
        return "OK"
    elif freshness_seconds < 3600:
        return "DEGRADED"
    else:
        return "STALE"


def determine_worst_status(statuses: list) -> str:
    """
    Return worst status from list.
    Priority: MISSING > STALE > DEGRADED > OK
    """
    priority = {"MISSING": 4, "STALE": 3, "DEGRADED": 2, "OK": 1}
    worst = max(statuses, key=lambda s: priority.get(s, 0))
    return worst
