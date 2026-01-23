#!/usr/bin/env python3
"""
Compare schemas between source and target databases.
Identifies missing tables, schema differences, and compatibility issues.
"""

import json
import sys
from datetime import datetime

import psycopg2
from psycopg2.extras import RealDictCursor


def get_all_tables(conn):
    """Get all user tables"""
    query = """
        SELECT
            schemaname,
            tablename
        FROM pg_tables
        WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'timescaledb_information')
        ORDER BY tablename;
    """
    cursor = conn.cursor(RealDictCursor)
    cursor.execute(query)
    tables = cursor.fetchall()
    return {f"{t['schemaname']}.{t['tablename']}" for t in tables}


def get_table_schema(conn, schema, table):
    """Get column definitions for table"""
    query = """
        SELECT
            column_name,
            data_type,
            character_maximum_length,
            numeric_precision,
            numeric_scale,
            is_nullable,
            column_default
        FROM information_schema.columns
        WHERE table_schema = %s AND table_name = %s
        ORDER BY ordinal_position;
    """
    cursor = conn.cursor(RealDictCursor)
    cursor.execute(query, (schema, table))
    return cursor.fetchall()


def get_primary_key(conn, schema, table):
    """Get primary key columns for table"""
    query = """
        SELECT a.attname
        FROM pg_index i
        JOIN pg_attribute a ON a.attrelid = i.indrelid
            AND a.attnum = ANY(i.indkey)
        WHERE i.indrelname = (
            SELECT indexname FROM pg_indexes
            WHERE schemaname = %s AND tablename = %s
            AND indexname LIKE '%pkey'
        )
        ORDER BY a.attnum;
    """
    cursor = conn.cursor()
    cursor.execute(query, (schema, table))
    pk_cols = [row[0] for row in cursor.fetchall()]
    return pk_cols if pk_cols else []


def format_column_type(col):
    """Format column type string"""
    dtype = col["data_type"]
    if col["character_maximum_length"]:
        dtype += f"({col['character_maximum_length']})"
    elif col["numeric_precision"]:
        dtype += f"({col['numeric_precision']},{col['numeric_scale']})"
    return dtype


def main():
    import os

    from dotenv import load_dotenv

    load_dotenv("/opt/.env")

    db_name = os.getenv("DB_NAME", "synctacles")
    db_user = os.getenv("DB_USER", "synctacles")

    # Source database config
    source_config = {
        "host": "localhost",
        "port": 5433,
        "database": db_name,
        "user": db_user,
        "connect_timeout": 10,
    }

    # Target database config
    target_config = {
        "host": "localhost",
        "port": 5432,
        "database": db_name,
        "user": db_user,
    }

    print("=" * 100)
    print("Schema Comparison: Source vs Target")
    print("=" * 100)
    print()

    # Connect to databases
    try:
        print(
            f"Connecting to source: {source_config['user']}@{source_config['host']}/{source_config['database']}...",
            end="",
            flush=True,
        )
        source_conn = psycopg2.connect(**source_config)
        print(" ✓")
    except Exception as e:
        print(" ✗")
        print(f"Error: {e}")
        sys.exit(1)

    try:
        print(
            f"Connecting to target: {target_config['user']}@{target_config['host']}/{target_config['database']}...",
            end="",
            flush=True,
        )
        target_conn = psycopg2.connect(**target_config)
        print(" ✓")
    except Exception as e:
        print(" ✗")
        print(f"Error: {e}")
        source_conn.close()
        sys.exit(1)

    print()

    # Get tables
    source_tables = get_all_tables(source_conn)
    target_tables = get_all_tables(target_conn)

    print(f"Source tables: {len(source_tables)}")
    print(f"Target tables: {len(target_tables)}")
    print()

    # Find differences
    missing_in_target = source_tables - target_tables
    extra_in_target = target_tables - source_tables
    common_tables = source_tables & target_tables

    comparison = {
        "timestamp": datetime.now().isoformat(),
        "source": f"{source_config['user']}@{source_config['host']}/{source_config['database']}",
        "target": f"{target_config['user']}@{target_config['host']}/{target_config['database']}",
        "summary": {
            "source_tables": len(source_tables),
            "target_tables": len(target_tables),
            "common_tables": len(common_tables),
            "missing_in_target": len(missing_in_target),
            "extra_in_target": len(extra_in_target),
        },
        "missing_in_target": list(missing_in_target),
        "extra_in_target": list(extra_in_target),
        "schema_mismatches": [],
    }

    # Check for missing tables
    if missing_in_target:
        print("⚠️  MISSING IN TARGET (will need to be created):")
        for table in sorted(missing_in_target):
            print(f"  - {table}")
        print()

    if extra_in_target:
        print("ℹ️  EXTRA IN TARGET (won't be migrated):")
        for table in sorted(extra_in_target):
            print(f"  - {table}")
        print()

    # Compare common tables
    if common_tables:
        print("=" * 100)
        print("Comparing Schema for Common Tables")
        print("=" * 100)
        print()

        all_match = True

        for table_name in sorted(common_tables):
            schema, table = table_name.split(".")

            source_schema = get_table_schema(source_conn, schema, table)
            target_schema = get_table_schema(target_conn, schema, table)

            if len(source_schema) != len(target_schema):
                all_match = False
                print(f"⚠️  {table_name}: Column count mismatch")
                print(f"  Source: {len(source_schema)} columns")
                print(f"  Target: {len(target_schema)} columns")
                comparison["schema_mismatches"].append(
                    {
                        "table": table_name,
                        "issue": "column_count_mismatch",
                        "source_columns": len(source_schema),
                        "target_columns": len(target_schema),
                    }
                )
                print()
                continue

            # Check each column
            columns_ok = True
            for i, (src_col, tgt_col) in enumerate(zip(source_schema, target_schema)):
                if src_col["column_name"] != tgt_col["column_name"]:
                    columns_ok = False
                    print(f"⚠️  {table_name}: Column name mismatch at position {i}")
                    print(f"  Source: {src_col['column_name']}")
                    print(f"  Target: {tgt_col['column_name']}")
                    comparison["schema_mismatches"].append(
                        {
                            "table": table_name,
                            "issue": "column_name_mismatch",
                            "position": i,
                        }
                    )

                if src_col["data_type"] != tgt_col["data_type"]:
                    columns_ok = False
                    print(
                        f"⚠️  {table_name}: Column type mismatch for '{src_col['column_name']}'"
                    )
                    print(f"  Source: {format_column_type(src_col)}")
                    print(f"  Target: {format_column_type(tgt_col)}")
                    comparison["schema_mismatches"].append(
                        {
                            "table": table_name,
                            "column": src_col["column_name"],
                            "issue": "type_mismatch",
                            "source_type": format_column_type(src_col),
                            "target_type": format_column_type(tgt_col),
                        }
                    )

            if columns_ok:
                print(f"✓ {table_name}: Schema matches")

        print()

    # Summary
    print("=" * 100)
    print("Summary")
    print("=" * 100)
    print(f"Common tables: {len(common_tables)}")
    print(f"Missing in target: {len(missing_in_target)}")
    print(f"Extra in target: {len(extra_in_target)}")
    print(f"Schema mismatches: {len(comparison['schema_mismatches'])}")
    print()

    if comparison["schema_mismatches"]:
        print("⚠️  WARNING: Schema mismatches found. Migration may require adjustments.")
    else:
        print("✓ All schemas compatible for migration!")

    # Save comparison to JSON
    comparison_file = (
        f"schema_comparison_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
    )
    with open(comparison_file, "w") as f:
        json.dump(comparison, f, indent=2, default=str)

    print()
    print(f"Detailed comparison saved to: {comparison_file}")
    print()

    source_conn.close()
    target_conn.close()


if __name__ == "__main__":
    main()
