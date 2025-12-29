# Database Migration Scripts

Production-ready bidirectional PostgreSQL database migration tools.

## Phase 1: Discovery & Analysis

Start with these scripts to understand the source database and verify connectivity.

### 1.1 Test Remote Connection

Auto-detect correct database name and credentials.

```bash
python test_remote_connection.py
```

**Output:**
- Working database name (synctacles or synctacles_db)
- Successfully authenticated user
- Database size and table count

**Use this info for subsequent scripts.**

### 1.2 Inventory Source Database

List all tables with row counts and sizes.

```bash
python inventory_source_database.py
```

**Modifies:** Update `config['database']` and `config['user']` based on Phase 1.1 results.

**Output:**
- Table listing with rows and sizes
- Total row count
- Saved to JSON: `inventory_synctacles_YYYYMMDD_HHMMSS.json`

### 1.3 Compare Schemas

Identify differences between source and target databases.

```bash
python compare_schemas.py
```

**Modifies:** Update source config based on Phase 1.1 results.

**Output:**
- Missing tables in target
- Extra tables in target
- Column type mismatches
- Saved to JSON: `schema_comparison_YYYYMMDD_HHMMSS.json`

## Phase 2: Migration

Once Phase 1 is complete and results reviewed.

### Main Migration Script

```bash
# Dry run (analysis without copying)
python migrate_database.py \
    --source synctacles \
    --target energy_insights_nl \
    --dry-run

# Migrate single table (test)
python migrate_database.py \
    --source synctacles \
    --target energy_insights_nl \
    --tables alembic_version

# Migrate all tables
python migrate_database.py \
    --source synctacles \
    --target energy_insights_nl
```

## Configuration

### Source Database (Remote)

Default: `synctacles@synctacles.com/synctacles`

Override with:
- `--source-host` - Remote host
- `--source-user` - Remote user
- `--source` - Database name

### Target Database (Local)

Default: `energy_insights_nl@localhost/energy_insights_nl`

Override with:
- `--target-host` - Local host
- `--target-user` - Local user
- `--target` - Database name

## Output

### Log Files

- `migration_YYYYMMDD_HHMMSS.log` - Detailed migration log
- `migration_report_YYYYMMDD_HHMMSS.json` - Structured results

### Examples

**migration_report.json:**
```json
{
  "timestamp": "2025-12-29T12:34:56",
  "source": "synctacles@synctacles.com/synctacles",
  "target": "energy_insights_nl@localhost/energy_insights_nl",
  "tables_migrated": [
    {
      "table": "public.alembic_version",
      "status": "success",
      "rows_copied": 1
    },
    {
      "table": "public.raw_prices",
      "status": "success",
      "rows_copied": 85000
    }
  ],
  "summary": {
    "successful_tables": 15,
    "failed_tables": 0,
    "total_tables": 15,
    "total_rows_migrated": 850000,
    "duration_seconds": 450
  }
}
```

## Workflow

### For Fresh Database Migration

1. **Run Phase 1.1:** Confirm connection works
   ```bash
   python test_remote_connection.py
   ```

2. **Update scripts:** Use discovered database name/user

3. **Run Phase 1.2:** Inventory source
   ```bash
   python inventory_source_database.py
   ```

4. **Run Phase 1.3:** Compare schemas
   ```bash
   python compare_schemas.py
   ```

5. **Review results:** Check inventory and schema comparison JSONs

6. **Run migration (dry):** Analyze without copying
   ```bash
   python migrate_database.py --source synctacles --target energy_insights_nl --dry-run
   ```

7. **Test single table:** Verify approach works
   ```bash
   python migrate_database.py --source synctacles --target energy_insights_nl --tables alembic_version
   ```

8. **Full migration:** When confident
   ```bash
   python migrate_database.py --source synctacles --target energy_insights_nl
   ```

### For Reverse Direction (Success Scenario)

To push local data back to production:

```bash
# Swap source and target
python migrate_database.py \
    --source energy_insights_nl \
    --target synctacles \
    --source-host localhost \
    --target-host synctacles.com \
    --tables raw_prices,norm_prices  # Specify which tables to sync
```

## Features

✅ **Bidirectional** - Works in both directions
✅ **Batched** - Processes data in configurable batches
✅ **Verified** - Reports row counts before/after
✅ **Logged** - Detailed logging to file and console
✅ **Safe** - Dry-run mode for analysis
✅ **Flexible** - Select specific tables or all tables

## Error Handling

- **Connection failed:** Check host, port, credentials
- **Table not found:** Run schema comparison first
- **Type mismatch:** May need schema updates before migration
- **Partial failure:** Review log file for specific table errors

## Performance

- Batch size: 1000 rows (configurable in code)
- Estimated speed: ~100k rows per minute
- Large migrations may take hours - run with appropriate timeframe

## Dependencies

```bash
pip install psycopg2-binary
```

## Notes

- Uses `ON CONFLICT DO NOTHING` to handle duplicates
- Preserves all data integrity (bytewise identical)
- Works with trust mode authentication
- Compatible with TimescaleDB
- Supports transactions (rollback on error per table)
