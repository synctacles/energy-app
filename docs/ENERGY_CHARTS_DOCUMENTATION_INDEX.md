# Energy-Charts API Documentation Index

Complete reference documentation for the Energy-Charts price API integration in SYNCTACLES.

**Last Updated:** 2026-01-02
**API Status:** Live and Verified
**Documentation Version:** 1.0

---

## Documentation Overview

This documentation package contains comprehensive information about the Energy-Charts API endpoint for Dutch electricity day-ahead prices, including:
- API response format specifications
- JSON schema for validation
- Complete implementation guide with code samples
- Real-world example responses with annotations

---

## Document Guide

### 1. Quick Start (5-10 min read)

**File:** [`ENERGY_CHARTS_API_SUMMARY.md`](./ENERGY_CHARTS_API_SUMMARY.md) (11 KB)

**Best for:** Getting oriented, understanding the basics

**Contains:**
- API endpoint URL and parameters
- Response structure overview
- Sample live data (348 price points)
- Python implementation pattern (15-line minimal example)
- Validation checklist
- Error handling quick reference
- Database schema
- API characteristics and comparisons

**Start here if you:** Need to understand the API quickly or are implementing integration

---

### 2. Complete API Reference (15-20 min read)

**File:** [`ENERGY_CHARTS_API_RESPONSE_FORMAT.md`](./ENERGY_CHARTS_API_RESPONSE_FORMAT.md) (9.9 KB)

**Best for:** Detailed understanding of response format

**Contains:**
- Request parameters (bzn, start, end)
- Complete response structure breakdown
- Field-by-field specifications:
  - `license_info` - licensing details
  - `unix_seconds` - timestamp array spec
  - `price` - price array spec
  - `unit` - measurement unit
  - `deprecated` - API status flag
- Complete example response
- Data characteristics (temporal, statistical)
- Data type specifications
- Validation rules and critical constraints
- Integration with SYNCTACLES (parsing, DB schema)
- Response validation checklist
- Error scenarios and handling
- API behavior notes (freshness, rate limiting)

**Start here if you:** Need comprehensive reference documentation

---

### 3. Implementation Guide (30-40 min read)

**File:** [`ENERGY_CHARTS_INTEGRATION_GUIDE.md`](./ENERGY_CHARTS_INTEGRATION_GUIDE.md) (15 KB)

**Best for:** Writing actual code

**Contains:**
- Current implementation status (collector, importer, client)
- Step-by-step parsing workflow:
  1. Fetch raw data
  2. Validate response structure
  3. Parse and transform data
  4. Store in database
- Complete code examples (fetch, validate, parse, store)
- Timestamp conversion patterns
- Database column definitions
- Error handling with fallback patterns
- Robust exception handling
- Unit test template with pytest
- Performance optimization tips
- Batch insert for efficiency
- Health check implementation
- Monitoring and alerting

**Start here if you:** Are writing or modifying code

---

### 4. JSON Schema (reference)

**File:** [`ENERGY_CHARTS_JSON_SCHEMA.json`](./ENERGY_CHARTS_JSON_SCHEMA.json) (3.2 KB)

**Best for:** Validation, automated testing, API documentation

**Contains:**
- JSON Schema Draft 7 format
- Type definitions for each field
- Required fields specification
- Array item type specifications
- Min/max constraints
- Enum values (unit)
- Example values
- Notes on array length matching
- Data constraints and timezone info

**Use this if you:**
- Need to validate responses programmatically
- Are setting up automated tests
- Want machine-readable schema

---

### 5. Annotated Example Response (reference)

**File:** [`ENERGY_CHARTS_RESPONSE_EXAMPLE.json`](./ENERGY_CHARTS_RESPONSE_EXAMPLE.json) (6.8 KB)

**Best for:** Understanding data mapping and relationships

**Contains:**
- Complete example response with inline comments
- Field-by-field explanation
- Timestamp conversion examples
- Data point alignment illustration
- Validation rules breakdown
- Parsing guidance
- Critical constraints highlighted
- Example data mapping (index to timestamp to price)

**Use this if you:**
- Want to understand real data structure
- Need to see example values with explanations
- Are debugging data processing

---

## File Relationships

```
ENERGY_CHARTS_API_SUMMARY.md (START HERE)
    ↓
    ├─→ ENERGY_CHARTS_API_RESPONSE_FORMAT.md (detailed reference)
    │       ↓
    │       └─→ ENERGY_CHARTS_JSON_SCHEMA.json (machine-readable)
    │
    ├─→ ENERGY_CHARTS_INTEGRATION_GUIDE.md (implementation)
    │       ↓
    │       └─→ Code examples, error handling, patterns
    │
    └─→ ENERGY_CHARTS_RESPONSE_EXAMPLE.json (example data)
            ↓
            └─→ Annotated with comments and mappings
```

---

## Quick Navigation

### By Task

**"I need to integrate this API"**
1. Read: `ENERGY_CHARTS_API_SUMMARY.md` (10 min)
2. Implement: Follow `ENERGY_CHARTS_INTEGRATION_GUIDE.md`
3. Reference: `ENERGY_CHARTS_API_RESPONSE_FORMAT.md`
4. Validate: Use `ENERGY_CHARTS_JSON_SCHEMA.json`

**"I need to debug a parsing issue"**
1. Check: `ENERGY_CHARTS_RESPONSE_EXAMPLE.json`
2. Reference: `ENERGY_CHARTS_API_RESPONSE_FORMAT.md` (field specs)
3. Validate: `ENERGY_CHARTS_JSON_SCHEMA.json`

**"I need to understand the API response"**
1. Quick: `ENERGY_CHARTS_API_SUMMARY.md` (overview)
2. Detailed: `ENERGY_CHARTS_API_RESPONSE_FORMAT.md` (full spec)
3. Example: `ENERGY_CHARTS_RESPONSE_EXAMPLE.json` (real data)

**"I need to write code"**
1. Pattern: `ENERGY_CHARTS_INTEGRATION_GUIDE.md` (code samples)
2. Reference: `ENERGY_CHARTS_API_RESPONSE_FORMAT.md`
3. Validate: `ENERGY_CHARTS_JSON_SCHEMA.json`
4. Test: `ENERGY_CHARTS_INTEGRATION_GUIDE.md` (test template)

---

## Key Concepts at a Glance

### Response Structure

The API returns a JSON object with 5 fields:

```json
{
  "license_info": "string",              // Data attribution
  "unix_seconds": [int, int, ...],       // Timestamps (UTC)
  "price": [float, float, ...],          // Prices (EUR/MWh)
  "unit": "EUR / MWh",                   // Always this
  "deprecated": false                    // API status
}
```

### Critical Rules

1. **Array Lengths Match:** `len(unix_seconds) == len(price)`
2. **Index Alignment:** `price[i]` ↔ `unix_seconds[i]`
3. **Time Interval:** 900 seconds (15 minutes) between timestamps
4. **Time Zone:** All timestamps are UTC
5. **Data Type:** Prices are floats, can have decimals

### Request Pattern

```bash
curl "https://api.energy-charts.info/price?bzn=NL&start=2025-12-31&end=2026-01-02"
```

### Processing Pattern

```python
# 1. Fetch
data = requests.get(URL, params=params).json()

# 2. Validate
assert len(data["unix_seconds"]) == len(data["price"])

# 3. Process
for unix_ts, price in zip(data["unix_seconds"], data["price"]):
    timestamp = datetime.fromtimestamp(unix_ts, tz=timezone.utc)
    # Store or process
```

---

## Current Implementation Status

### Files in Project

| File | Purpose | Status |
|------|---------|--------|
| `/synctacles_db/collectors/energy_charts_prices.py` | Fetches raw JSON | ✓ Implemented |
| `/synctacles_db/importers/import_energy_charts_prices.py` | Parses and imports | ✓ Implemented |
| `/synctacles_db/fallback/energy_charts_client.py` | Fallback client | ✓ Has generation mix; needs price support |

### What's Documented Here

These documentation files detail the API response format that `fetch_prices()` in `energy_charts_client.py` should parse.

### Next Steps

1. **Add price endpoint** to EnergyChartsClient class
2. **Implement error handling** with circuit breaker
3. **Add unit tests** using examples
4. **Integrate with fallback** system
5. **Add monitoring** for data freshness

---

## API Specifications Summary

| Aspect | Details |
|--------|---------|
| **Endpoint** | `https://api.energy-charts.info/price` |
| **Method** | GET |
| **Authentication** | None (public API) |
| **Rate Limit** | 10 requests/minute (free tier) |
| **Response Format** | JSON |
| **Response Time** | ~330ms typical |
| **Timeout** | Set to 30 seconds recommended |
| **Data Source** | SMARD.de (APX ENDEX market) |
| **Data Frequency** | 15-minute intervals |
| **Market Type** | Day-ahead auction |
| **Update Time** | ~12:42 CET daily |
| **Historical Data** | Several years available |
| **Time Zone** | UTC |
| **Array Length** | Varies (typically 280-360 for 2-day request) |

---

## Validation Checklist

Before trusting price data:

- [ ] HTTP status 200 (success)
- [ ] JSON parses without error
- [ ] All 5 required fields present
- [ ] `unix_seconds` is array of integers
- [ ] `price` is array of numbers
- [ ] Array lengths match exactly
- [ ] Timestamps in ascending order
- [ ] Timestamp increment is ~900 seconds
- [ ] `unit` is "EUR / MWh"
- [ ] `deprecated` is false

---

## Error Handling Reference

| Error | Cause | Fix |
|-------|-------|-----|
| `KeyError: 'unix_seconds'` | Missing field | Validate structure first |
| Length mismatch | Corrupted data | Check both arrays same length |
| Invalid timestamp | Timezone issue | Use `tz=timezone.utc` |
| Type error in price | Non-numeric value | Validate array contains numbers |
| HTTP 429 | Rate limited | Wait 6 seconds, implement cache |
| HTTP 404 | No data for dates | Verify date range valid |
| HTTP 500 | Server error | Retry with backoff |

---

## Code Examples Location

See `ENERGY_CHARTS_INTEGRATION_GUIDE.md` for:
- Fetch implementation (request building, error handling)
- Validation function (field and array checking)
- Parsing function (timestamp conversion, data transformation)
- Database import function (SQL insert patterns)
- Unit test template (pytest examples)
- Error handling patterns (exception handling, fallback)
- Performance tips (batch insert, caching)

---

## Testing

### Manual API Test

```bash
curl -i "https://api.energy-charts.info/price?bzn=NL&start=2025-12-31&end=2026-01-02"
```

Expected:
- HTTP 200
- Valid JSON
- Arrays of equal length
- Timestamps increment by 900

### Unit Test Template

See `ENERGY_CHARTS_INTEGRATION_GUIDE.md` for complete pytest examples

---

## Related Documentation

- **Data Sources Master:** `/docs/skills/SKILL_06_DATA_SOURCES.md`
- **Architecture:** `/docs/ARCHITECTURE.md`
- **Collector Code:** `/synctacles_db/collectors/energy_charts_prices.py`
- **Importer Code:** `/synctacles_db/importers/import_energy_charts_prices.py`

---

## Document Statistics

| Document | Size | Content |
|----------|------|---------|
| Summary | 11 KB | Quick ref + minimal example |
| API Format | 9.9 KB | Complete field specifications |
| Integration Guide | 15 KB | Full code examples + tests |
| JSON Schema | 3.2 KB | Machine-readable format |
| Example Response | 6.8 KB | Live data with annotations |
| **Total** | **46 KB** | Complete reference suite |

---

## Verification

**API Status:** ✓ Live and Verified
**Last Verified:** 2026-01-02
**Response:** Valid JSON, 348 price points, 5,020 bytes

**Sample Data:**
- Date Range: 2025-12-31 to 2026-01-02
- Time Interval: 15 minutes
- Price Range: 2.34 - 119.98 EUR/MWh
- Source: SMARD.de (APX ENDEX market)

---

## How to Use This Documentation

1. **First Time:** Read `ENERGY_CHARTS_API_SUMMARY.md`
2. **Deep Dive:** Read `ENERGY_CHARTS_API_RESPONSE_FORMAT.md`
3. **Implementation:** Follow `ENERGY_CHARTS_INTEGRATION_GUIDE.md`
4. **Reference:** Use other docs as needed
5. **Validation:** Use `ENERGY_CHARTS_JSON_SCHEMA.json`
6. **Examples:** Check `ENERGY_CHARTS_RESPONSE_EXAMPLE.json`

---

## Questions?

- **What's the API response format?** → See `ENERGY_CHARTS_API_RESPONSE_FORMAT.md`
- **How do I implement it?** → See `ENERGY_CHARTS_INTEGRATION_GUIDE.md`
- **What are the field specs?** → See `ENERGY_CHARTS_API_RESPONSE_FORMAT.md` (field descriptions) or JSON Schema
- **Can I validate it?** → Use `ENERGY_CHARTS_JSON_SCHEMA.json`
- **What's an example?** → See `ENERGY_CHARTS_RESPONSE_EXAMPLE.json`
- **Is the API live?** → Yes, verified 2026-01-02

---

## Version History

| Date | Version | Status |
|------|---------|--------|
| 2026-01-02 | 1.0 | Initial complete documentation with live API verification |

---

**Documentation Created By:** Claude Code Assistant
**Last Updated:** 2026-01-02
**Next Review:** When API changes or new features added
