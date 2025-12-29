#!/usr/bin/env python3
"""
Inventory source database: List all tables with row counts and sizes.
"""

import psycopg2
from psycopg2.extras import RealDictCursor
import sys
import json
from datetime import datetime


def get_all_tables(conn):
    """Get all user tables (excluding system tables)"""
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
    return cursor.fetchall()


def get_table_stats(conn, schema, table):
    """Get row count and size for table"""
    try:
        cursor = conn.cursor()

        # Row count (exact)
        cursor.execute(f"SELECT COUNT(*) FROM {schema}.{table};")
        row_count = cursor.fetchone()[0]

        # Size
        cursor.execute(
            f"SELECT pg_size_pretty(pg_total_relation_size('{schema}.{table}')) as size;"
        )
        size = cursor.fetchone()[0]

        # Size in bytes
        cursor.execute(
            f"SELECT pg_total_relation_size('{schema}.{table}') as bytes;"
        )
        size_bytes = cursor.fetchone()[0]

        return {
            'row_count': row_count,
            'size': size,
            'size_bytes': size_bytes,
        }
    except Exception as e:
        return {
            'row_count': 0,
            'size': 'ERROR',
            'size_bytes': 0,
            'error': str(e)
        }


def main():
    # Connection config - modify these based on test_remote_connection.py results
    config = {
        'host': 'localhost',
        'port': 5433,
        'database': 'synctacles',  # Change if needed
        'user': 'synctacles',  # Change if needed
        'connect_timeout': 10
    }

    print("=" * 100)
    print("Source Database Inventory")
    print("=" * 100)
    print()

    try:
        print(f"Connecting to {config['user']}@{config['host']}/{config['database']}...", flush=True)
        conn = psycopg2.connect(**config)

        cursor = conn.cursor()
        cursor.execute(f"SELECT pg_size_pretty(pg_database_size('{config['database']}'));")
        db_size = cursor.fetchone()[0]
        print(f"✓ Connected (database size: {db_size})")
        print()

    except Exception as e:
        print(f"✗ Failed: {e}")
        print()
        print("Run test_remote_connection.py first to find working configuration")
        sys.exit(1)

    # Get all tables
    try:
        tables = get_all_tables(conn)
        print(f"Found {len(tables)} tables")
        print()
    except Exception as e:
        print(f"✗ Failed to list tables: {e}")
        sys.exit(1)

    # Collect statistics
    print(f"{'Schema':<15} {'Table':<40} {'Rows':>15} {'Size':>15}")
    print("-" * 100)

    inventory = []
    total_rows = 0
    total_size_bytes = 0

    for table_info in tables:
        schema = table_info['schemaname']
        table = table_info['tablename']

        stats = get_table_stats(conn, schema, table)
        row_count = stats['row_count']
        size = stats['size']
        size_bytes = stats['size_bytes']

        total_rows += row_count
        total_size_bytes += size_bytes

        print(f"{schema:<15} {table:<40} {row_count:>15,} {size:>15}")

        inventory.append({
            'schema': schema,
            'table': table,
            'row_count': row_count,
            'size': size,
            'size_bytes': size_bytes,
        })

    print("-" * 100)
    print(f"{'TOTAL':<15} {'':<40} {total_rows:>15,} {total_size_bytes / (1024**3):>14.2f} GB")
    print()

    # Summary
    print("=" * 100)
    print("Summary")
    print("=" * 100)
    print(f"Total tables: {len(tables)}")
    print(f"Total rows: {total_rows:,}")
    print(f"Total size: {total_size_bytes / (1024**3):.2f} GB")
    print()

    # Save inventory to JSON
    inventory_file = f"inventory_{config['database']}_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
    with open(inventory_file, 'w') as f:
        json.dump({
            'timestamp': datetime.now().isoformat(),
            'source': f"{config['user']}@{config['host']}/{config['database']}",
            'tables': inventory,
            'summary': {
                'total_tables': len(tables),
                'total_rows': total_rows,
                'total_size_bytes': total_size_bytes,
                'total_size_gb': total_size_bytes / (1024**3),
            }
        }, f, indent=2, default=str)

    print(f"Inventory saved to: {inventory_file}")
    print()

    conn.close()


if __name__ == '__main__':
    main()
