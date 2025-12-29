#!/usr/bin/env python3
"""
Test remote PostgreSQL database connection.
Auto-detects which database name and user credentials work.
"""

import psycopg2
import sys
from tabulate import tabulate

# Try all possible database/user combinations
possible_configs = [
    {'host': 'localhost', 'port': 5433, 'database': 'synctacles', 'user': 'synctacles'},
]

print("=" * 80)
print("Testing Remote Database Connections")
print("=" * 80)
print()

successful_config = None

for config in possible_configs:
    config_str = f"{config['user']}@{config['host']}/{config['database']}"
    print(f"Testing: {config_str:<50} ", end='', flush=True)

    try:
        conn = psycopg2.connect(**config, connect_timeout=5)

        cursor = conn.cursor()

        # Get database info
        cursor.execute("""
            SELECT
                current_database() as db_name,
                pg_size_pretty(pg_database_size(current_database())) as size,
                current_user as user;
        """)
        db_name, size, user = cursor.fetchone()

        # Count tables
        cursor.execute("""
            SELECT COUNT(*) FROM information_schema.tables
            WHERE table_schema NOT IN ('pg_catalog', 'information_schema');
        """)
        table_count = cursor.fetchone()[0]

        # Get version
        cursor.execute("SELECT version();")
        version = cursor.fetchone()[0].split(',')[0]

        print(f"✓ SUCCESS")
        print(f"  ├─ Database: {db_name}")
        print(f"  ├─ User: {user}")
        print(f"  ├─ Size: {size}")
        print(f"  ├─ Tables: {table_count}")
        print(f"  └─ Version: {version}")
        print()

        successful_config = config
        conn.close()

    except psycopg2.OperationalError as e:
        error_msg = str(e).split('\n')[0][:40]
        print(f"✗ Failed: {error_msg}")
    except Exception as e:
        error_msg = str(e).split('\n')[0][:40]
        print(f"✗ Error: {error_msg}")

print()
print("=" * 80)

if successful_config:
    print("SUCCESS! Working Configuration Found:")
    print()
    print(f"  Host:     {successful_config['host']}")
    print(f"  Port:     {successful_config['port']}")
    print(f"  Database: {successful_config['database']}")
    print(f"  User:     {successful_config['user']}")
    print()
    print("Use this configuration in migrate_database.py")
    print()
    sys.exit(0)
else:
    print("FAILED: Could not connect to any database configuration!")
    print()
    print("Check:")
    print("  - Remote host is reachable (ping synctacles.com)")
    print("  - PostgreSQL is running on remote server")
    print("  - Firewall allows port 5432")
    print("  - Database name is correct")
    print("  - User credentials are correct")
    print()
    sys.exit(1)
