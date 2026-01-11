# ADR-003: TenneT Data Exclusion from Coefficient Calculation

**Date:** 2026-01-10
**Status:** Accepted
**Deciders:** Leo, Claude Code, CAI (Claude AI)
**Related:** ADR-001 (TenneT BYO-Key), HANDOFF_CC_COEFFICIENT_ENGINE.md
**Scope:** Coefficient Engine Data Model

---

## Context

The coefficient engine calculates a multiplier that converts wholesale electricity prices (ENTSO-E) to estimated consumer prices. The formula is:

```
coefficient = consumer_price (Enever) / wholesale_price (ENTSO-E)
estimated_consumer_price = wholesale_price × coefficient
```

The question arose: **Should TenneT grid balance data be included in the coefficient calculation?**

TenneT provides:
- Real-time grid balance (generation vs. load)
- Grid congestion indicators
- Imbalance metrics

These factors could potentially correlate with price variations.

---

## Decision

**TenneT data is NOT included in the initial coefficient calculation (Phase 1).**

The coefficient engine Phase 1 uses only:
- ✅ **Enever** consumer prices (requires NL IP, via VPN)
- ✅ **ENTSO-E** wholesale prices (no IP restriction)

TenneT data MAY be evaluated as an optional refinement in Phase 2, but only if:
1. Basic coefficient proves unstable (high variance)
2. Analysis shows significant correlation between grid metrics and coefficient variance
3. The predictive value justifies the added complexity

---

## Rationale

### Why TenneT is Not in the Formula

The coefficient formula is:
```
coefficient = consumer_price / wholesale_price
```

**TenneT provides neither consumer prices nor wholesale prices.** It provides grid operation data (balance, load, generation).

### Could TenneT Data Improve the Model?

**Hypothesis (CAI):** There might be a correlation between grid stress and price markup:

| TenneT Metric | Potential Effect on Coefficient |
|---------------|----------------------------------|
| High grid imbalance | → Higher consumer markup? |
| Grid congestion | → Price spikes at consumer level? |
| Extreme load | → Larger spread wholesale ↔ consumer? |

**Example Scenario:**
```
Normal conditions:     coefficient = 1.25
Grid stress detected:  coefficient = 1.35 (consumers pay extra?)
```

**Analysis Required:**
```sql
-- Check correlation between grid balance and coefficient variance
SELECT
    DATE_TRUNC('hour', e.timestamp) AS hour,
    AVG(coefficient) AS avg_coefficient,
    AVG(t.balance_delta) AS avg_balance,
    CORR(coefficient, t.balance_delta) AS correlation
FROM coefficient_data c
JOIN tennet_balance t ON c.timestamp = t.timestamp
GROUP BY 1
```

**Reality:** We don't know if this correlation exists.

Possible outcomes:
- **Strong correlation**: TenneT becomes a factor in the model
  → `coefficient = base_coefficient × grid_stress_factor`
- **Weak correlation**: TenneT only used for Grid Stress sensor (ADR-001 use case)
- **No correlation**: Ignore TenneT for coefficient entirely

---

## Alternatives Considered

### 1. Include TenneT from Phase 1 (REJECTED)

**Approach:** Fetch TenneT data immediately and include in coefficient calculation.

**Pros:**
- Potentially more accurate coefficient
- Captures grid dynamics

**Cons:**
- ❌ No evidence correlation exists
- ❌ Adds complexity before validating basic model
- ❌ TenneT API requires separate auth/management
- ❌ Violates "simplest thing that works" principle
- ❌ Delays Phase 1 completion (blocked on TenneT access)

**Why rejected:** Build the foundation first. Measure whether basic coefficient is stable. Only add complexity if data shows it's needed.

### 2. TenneT as Primary Data Source (REJECTED)

**Approach:** Use TenneT balance to predict consumer prices directly.

**Cons:**
- ❌ TenneT doesn't provide price data
- ❌ Correlation between balance and consumer prices is unproven
- ❌ Would require complex modeling (ML?)
- ❌ Much harder to validate

**Why rejected:** TenneT is the wrong data source for price modeling. Enever provides actual consumer prices, which is direct and reliable.

### 3. Always Include TenneT (Even Without Correlation) (REJECTED)

**Approach:** Add TenneT to the model regardless of correlation.

**Cons:**
- ❌ Adding noise without signal degrades model quality
- ❌ Over-engineering without evidence
- ❌ Maintenance burden for no benefit

**Why rejected:** Data-driven approach requires evidence before adding features.

---

## Phase 1 Decision: Build Without TenneT

### What Phase 1 Includes

```python
# Basic coefficient calculation
coefficient = enever_consumer_price / entso_e_wholesale_price

# Example:
# Enever: €0.25/kWh (consumer pays)
# ENTSO-E: €200/MWh = €0.20/kWh (wholesale)
# Coefficient: 0.25 / 0.20 = 1.25
```

### Stability Analysis (Phase 1)

After collecting Enever + ENTSO-E data, analyze coefficient stability:

```python
# analysis/stability_report.py
def stability_report(df):
    # Calculate variance
    avg_std = df['coefficient'].std()
    avg_mean = df['coefficient'].mean()
    cv = (avg_std / avg_mean) * 100  # Coefficient of variation

    if cv < 10:
        print("✅ STABLE - Lookup table sufficient")
        # Use simple lookup: coefficient by month/hour/day_type
    else:
        print("⚠️ UNSTABLE - Investigate refinements")
        # Proceed to Phase 2: analyze TenneT correlation
```

**If stable (CV < 10%):**
- Use lookup table approach
- No need for TenneT
- Coefficient varies predictably by time patterns

**If unstable (CV > 10%):**
- Investigate causes of variance
- Check TenneT correlation as one hypothesis
- May need real-time calculation

---

## Phase 2: TenneT as Optional Refinement

**Status:** DEFERRED until Phase 1 analysis complete

**When to Consider TenneT:**
1. ✅ Enever + ENTSO-E data collected (3+ months minimum)
2. ✅ Basic coefficient calculated
3. ✅ Stability analysis shows CV > 10% (unstable)
4. ✅ TenneT access available (API key obtained)

**How to Evaluate:**

```python
# 1. Collect TenneT data for same time period
# 2. Calculate correlation
correlation = analyze_correlation(
    coefficients=df['coefficient'],
    balance_delta=tennet_df['balance_delta'],
    congestion=tennet_df['congestion_indicator']
)

# 3. Decide based on correlation strength
if abs(correlation) > 0.5:  # Strong correlation
    print("TenneT data is predictive - add to model")
elif abs(correlation) > 0.3:  # Moderate
    print("TenneT may help - A/B test refined model")
else:  # Weak
    print("TenneT not predictive - keep simple model")
```

**If TenneT is predictive:**

```python
# Refined coefficient model
base_coefficient = lookup_coefficient(month, hour, day_type)

# Grid stress factor from TenneT
grid_stress = calculate_grid_stress(balance_delta, congestion)

# Final coefficient
coefficient = base_coefficient * (1 + grid_stress_adjustment)
```

---

## Consequences

### Positive

✅ **Faster Time to Market**: Phase 1 unblocked, no TenneT dependency
✅ **Simpler Initial Model**: Easier to debug and validate
✅ **Data-Driven Decisions**: Add TenneT only if data supports it
✅ **Lower Maintenance**: Fewer data sources = fewer failure modes
✅ **VPN Scope Reduced**: Only Enever needs NL IP (not TenneT)

### Negative

⚠️ **Potentially Less Accurate**: If grid stress DOES affect prices significantly
- **Mitigation**: Phase 2 can add TenneT if analysis proves valuable
- **Impact**: Low - coefficient still based on actual Enever prices

⚠️ **May Need Future Refactoring**: Adding TenneT in Phase 2 requires code changes
- **Mitigation**: Design allows for `grid_stress_factor` to be added modularly
- **Impact**: Low - architecture designed for extensibility

### Neutral

ℹ️ **TenneT Still Valuable for Grid Stress Sensor**: ADR-001 use case unaffected
- Users can still get grid balance via BYO-key in HA component
- Coefficient engine and Grid Stress sensor are separate concerns

---

## Testing Strategy

### Phase 1 Validation

After Enever + ENTSO-E data collection:

```bash
# 1. Generate stability report
python analysis/stability_report.py

# 2. Visual inspection
python analysis/plot_coefficient_variance.py

# 3. Statistical tests
python analysis/coefficient_statistics.py
```

**Success Criteria (Phase 1):**
- Coefficient varies predictably by time-of-day
- CV < 10% (stable)
- Lookup table provides reasonable estimates

**If success criteria not met:**
- Investigate variance causes
- Evaluate TenneT correlation (Phase 2)
- Consider other factors (weather, holidays, etc.)

### Phase 2 Evaluation (If Needed)

```python
# Add TenneT data for same period
# Calculate correlation
# A/B test: simple vs. TenneT-enhanced model
# Measure improvement in prediction accuracy
# Decide: keep simple or adopt enhanced model
```

---

## Architecture Impact

### Current Architecture (Phase 1)

```
┌─────────────────────────────────────────┐
│  COEFFICIENT SERVER                     │
│  ├── Enever Collector (via NL VPN)      │
│  ├── ENTSO-E Collector (direct)         │
│  ├── Coefficient Calculator             │
│  │   coefficient = Enever / ENTSO-E     │
│  └── API: GET /coefficient              │
└─────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────┐
│  SYNCTACLES SERVER                      │
│  ├── Fetches coefficient                │
│  ├── Calculates Energy Action           │
│  └── Serves to HA components            │
└─────────────────────────────────────────┘
```

### Potential Phase 2 Architecture (If TenneT Proves Valuable)

```
┌─────────────────────────────────────────┐
│  COEFFICIENT SERVER                     │
│  ├── Enever Collector (via NL VPN)      │
│  ├── ENTSO-E Collector (direct)         │
│  ├── TenneT Collector (via NL VPN)      │ ← Added
│  ├── Enhanced Coefficient Calculator    │ ← Modified
│  │   base = Enever / ENTSO-E            │
│  │   stress = f(TenneT balance)         │
│  │   coefficient = base × (1 + stress)  │
│  └── API: GET /coefficient              │
└─────────────────────────────────────────┘
```

**Note:** Architecture allows for this evolution without breaking changes.

---

## Documentation Updates Required

When TenneT is evaluated in Phase 2:

1. **Update HANDOFF_CC_COEFFICIENT_ENGINE.md**
   - Add TenneT correlation analysis results
   - Document decision to include/exclude TenneT
   - Update coefficient formula if refined

2. **Update SKILL_10 (if TenneT added)**
   - Add TenneT collector setup
   - VPN routing for TenneT API (if NL IP required)
   - Troubleshooting TenneT-related issues

3. **Update ARCHITECTURE.md**
   - Add TenneT to data flow diagram
   - Document grid stress factor calculation
   - Explain refined coefficient model

---

## Decision Timeline

| Phase | Action | Status |
|-------|--------|--------|
| Phase 1 | Implement Enever + ENTSO-E coefficient | ⏳ IN PROGRESS |
| Phase 1 | Collect 3-6 months of data | ⏳ PENDING |
| Phase 1 | Generate stability report | ⏳ PENDING |
| Phase 1 | Decide: stable or unstable? | ⏳ PENDING |
| Phase 2 | If unstable: collect TenneT data | 🔒 CONDITIONAL |
| Phase 2 | Analyze TenneT correlation | 🔒 CONDITIONAL |
| Phase 2 | Decision: add TenneT or keep simple | 🔒 CONDITIONAL |

**Earliest Phase 2 decision:** 3-6 months after Phase 1 launch

---

## Success Metrics

### Phase 1 Success

✅ Coefficient calculated from Enever + ENTSO-E
✅ Stability analysis complete
✅ Coefficient API functional
✅ SYNCTACLES integration working
✅ Energy Action using coefficient-based estimates

### Phase 2 Success (If Pursued)

✅ TenneT correlation quantified
✅ Data-driven decision on inclusion
✅ If included: improved prediction accuracy
✅ If excluded: documentation of analysis for future reference

---

## Related Decisions

- **ADR-001**: TenneT BYO-Key model for real-time balance data
  - TenneT is available for Grid Stress sensors
  - Separate concern from coefficient calculation

- **ADR-002**: VPN split tunneling
  - Currently only routes Enever traffic
  - Could extend to TenneT in Phase 2 if needed

---

## References

- CAI Analysis: TenneT correlation hypothesis (2026-01-10 conversation)
- HANDOFF_CC_COEFFICIENT_ENGINE.md: Coefficient formula and architecture
- ADR-001: TenneT BYO-Key decision

---

## Changelog

- **2026-01-10**: Initial decision (Claude Code + CAI)
  - TenneT excluded from Phase 1
  - Phase 2 evaluation path defined
  - Stability analysis criteria established
