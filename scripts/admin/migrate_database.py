#!/usr/bin/env python3
"""
Bidirectional PostgreSQL Database Migration Script
Migrate data between any two PostgreSQL databases with batching, verification, and logging.
"""

import argparse
import json
import logging
import sys
from datetime import datetime

import psycopg2
from psycopg2.extras import RealDictCursor


class DatabaseMigration:
    """Handle database migration operations"""

    def __init__(self, source_config: dict, target_config: dict, dry_run: bool = False):
        self.source_config = source_config
        self.target_config = target_config
        self.dry_run = dry_run
        self.source_conn = None
        self.target_conn = None
        self.migration_log = {
            'timestamp': datetime.now().isoformat(),
            'dry_run': dry_run,
            'source': f"{source_config['user']}@{source_config.get('host', 'localhost')}/{source_config['database']}",
            'target': f"{target_config['user']}@{target_config.get('host', 'localhost')}/{target_config['database']}",
            'tables_migrated': [],
            'errors': [],
            'summary': {}
        }

        # Setup logging
        log_file = f"migration_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log"
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(log_file),
                logging.StreamHandler()
            ]
        )
        self.logger = logging.getLogger(__name__)
        self.logger.info(f"Migration started (dry_run={dry_run})")

    def connect(self):
        """Establish connections to both databases"""
        self.logger.info(f"Connecting to source: {self.source_config['database']}...")
        try:
            self.source_conn = psycopg2.connect(**self.source_config)
            self.logger.info("✓ Connected to source")
        except Exception as e:
            self.logger.error(f"✗ Failed to connect to source: {e}")
            sys.exit(1)

        self.logger.info(f"Connecting to target: {self.target_config['database']}...")
        try:
            self.target_conn = psycopg2.connect(**self.target_config)
            self.logger.info("✓ Connected to target")
        except Exception as e:
            self.logger.error(f"✗ Failed to connect to target: {e}")
            if self.source_conn:
                self.source_conn.close()
            sys.exit(1)

    def disconnect(self):
        """Close database connections"""
        if self.source_conn:
            self.source_conn.close()
        if self.target_conn:
            self.target_conn.close()

    def get_all_tables(self, conn) -> list[tuple[str, str]]:
        """Get list of all user tables"""
        query = """
            SELECT schemaname, tablename
            FROM pg_tables
            WHERE schemaname = 'public'
            ORDER BY tablename;
        """
        cursor = conn.cursor(cursor_factory=RealDictCursor)
        cursor.execute(query)
        tables = cursor.fetchall()
        return [(t['schemaname'], t['tablename']) for t in tables]

    def get_row_count(self, conn, schema: str, table: str) -> int:
        """Get row count for table"""
        try:
            cursor = conn.cursor()
            cursor.execute(f"SELECT COUNT(*) FROM {schema}.{table};")
            return cursor.fetchone()[0]
        except Exception as e:
            self.logger.warning(f"Failed to get row count for {schema}.{table}: {e}")
            return 0

    def get_table_columns(self, conn, schema: str, table: str) -> list[str]:
        """Get ordered list of column names"""
        cursor = conn.cursor(cursor_factory=RealDictCursor)
        query = """
            SELECT column_name
            FROM information_schema.columns
            WHERE table_schema = %s AND table_name = %s
            ORDER BY ordinal_position;
        """
        cursor.execute(query, (schema, table))
        return [row['column_name'] for row in cursor.fetchall()]

    def copy_table(self, schema: str, table: str, batch_size: int = 1000) -> dict:
        """Copy table data in batches"""
        table_name = f"{schema}.{table}"
        self.logger.info(f"Migrating {table_name}...")

        # Get row counts
        source_count = self.get_row_count(self.source_conn, schema, table)
        target_count = self.get_row_count(self.target_conn, schema, table)

        self.logger.info(f"  Source: {source_count:,} rows | Target: {target_count:,} rows")

        # Get columns
        columns = self.get_table_columns(self.source_conn, schema, table)
        columns_str = ', '.join(columns)

        # Fetch and copy in batches
        cursor = self.source_conn.cursor(cursor_factory=RealDictCursor)
        cursor.itersize = batch_size

        query = f"SELECT {columns_str} FROM {table_name} ORDER BY {columns[0]} ASC"

        try:
            cursor.execute(query)
        except Exception as e:
            self.logger.error(f"  ✗ Failed to query source: {e}")
            return {
                'table': table_name,
                'status': 'error',
                'error': str(e),
                'rows_copied': 0
            }

        rows_copied = 0
        batches_processed = 0

        try:
            while True:
                rows = cursor.fetchmany(batch_size)
                if not rows:
                    break

                if not self.dry_run:
                    # Prepare insert statement
                    placeholders = ', '.join(['%s'] * len(columns))
                    insert_query = f"INSERT INTO {table_name} ({columns_str}) VALUES ({placeholders}) ON CONFLICT DO NOTHING"

                    target_cursor = self.target_conn.cursor()
                    for row in rows:
                        values = tuple(row[col] for col in columns)
                        try:
                            target_cursor.execute(insert_query, values)
                        except Exception as e:
                            self.logger.warning(f"  Failed to insert row: {e}")

                    target_cursor.close()
                    self.target_conn.commit()

                rows_copied += len(rows)
                batches_processed += 1

                # Progress update every 10 batches
                if batches_processed % 10 == 0:
                    self.logger.info(f"  Progress: {rows_copied:,} rows...")

        except Exception as e:
            self.logger.error(f"  ✗ Migration failed: {e}")
            return {
                'table': table_name,
                'status': 'error',
                'error': str(e),
                'rows_copied': rows_copied
            }

        cursor.close()

        self.logger.info(f"  ✓ Migrated {rows_copied:,} rows")

        # Verify
        target_count_after = self.get_row_count(self.target_conn, schema, table)
        self.logger.info(f"  Target now has: {target_count_after:,} rows")

        return {
            'table': table_name,
            'status': 'success',
            'source_rows': source_count,
            'target_rows_before': target_count,
            'rows_copied': rows_copied,
            'target_rows_after': target_count_after,
        }

    def analyze(self):
        """Analyze migration without executing"""
        self.logger.info("=" * 80)
        self.logger.info("DRY RUN: Analysis Only")
        self.logger.info("=" * 80)

        # Get tables
        source_tables = self.get_all_tables(self.source_conn)
        target_tables = self.get_all_tables(self.target_conn)

        self.logger.info(f"Source tables: {len(source_tables)}")
        self.logger.info(f"Target tables: {len(target_tables)}")

        # Calculate totals
        total_rows = 0
        total_size = 0

        self.logger.info("")
        self.logger.info("Table Analysis:")
        self.logger.info(f"{'Table':<40} {'Rows':>15} {'Size':>15}")
        self.logger.info("-" * 70)

        for schema, table in source_tables:
            table_name = f"{schema}.{table}"
            row_count = self.get_row_count(self.source_conn, schema, table)
            total_rows += row_count

            cursor = self.source_conn.cursor()
            cursor.execute(f"SELECT pg_total_relation_size('{table_name}') as bytes;")
            size_bytes = cursor.fetchone()[0]
            total_size += size_bytes

            size_mb = size_bytes / (1024 * 1024)
            self.logger.info(f"{table:<40} {row_count:>15,} {size_mb:>14.2f} MB")

        self.logger.info("-" * 70)
        self.logger.info(f"{'TOTAL':<40} {total_rows:>15,} {total_size / (1024**3):>14.2f} GB")
        self.logger.info("")

        # Estimates
        minutes_estimate = (total_rows / 100000) * 2  # Very rough estimate
        self.logger.info(f"Estimated migration time: ~{int(minutes_estimate)} minutes")
        self.logger.info(f"Estimated disk space needed: {total_size / (1024**3) * 2:.2f} GB (2x for safety)")
        self.logger.info("")

    def execute(self, table_filter: str = None):
        """Execute migration"""
        self.logger.info("=" * 80)
        self.logger.info("EXECUTING MIGRATION")
        self.logger.info("=" * 80)

        source_tables = self.get_all_tables(self.source_conn)

        # Filter tables if requested
        if table_filter:
            table_names = [t.strip() for t in table_filter.split(',')]
            source_tables = [(s, t) for s, t in source_tables if t in table_names]
            self.logger.info(f"Filtering to tables: {', '.join(table_names)}")

        self.logger.info(f"Migrating {len(source_tables)} tables...")
        self.logger.info("")

        start_time = datetime.now()

        for schema, table in source_tables:
            result = self.copy_table(schema, table)
            self.migration_log['tables_migrated'].append(result)

            if result['status'] == 'error':
                self.migration_log['errors'].append({
                    'table': result['table'],
                    'error': result['error']
                })

        end_time = datetime.now()
        duration = (end_time - start_time).total_seconds()

        self.logger.info("")
        self.logger.info("=" * 80)
        self.logger.info("Migration Complete")
        self.logger.info("=" * 80)

        # Generate summary
        successful = sum(1 for t in self.migration_log['tables_migrated'] if t['status'] == 'success')
        failed = sum(1 for t in self.migration_log['tables_migrated'] if t['status'] == 'error')
        total_rows = sum(t.get('rows_copied', 0) for t in self.migration_log['tables_migrated'])

        self.migration_log['summary'] = {
            'successful_tables': successful,
            'failed_tables': failed,
            'total_tables': len(self.migration_log['tables_migrated']),
            'total_rows_migrated': total_rows,
            'duration_seconds': duration,
        }

        self.logger.info(f"Tables migrated: {successful}/{len(source_tables)}")
        self.logger.info(f"Total rows: {total_rows:,}")
        self.logger.info(f"Duration: {duration:.1f} seconds")

        if failed > 0:
            self.logger.warning(f"Failures: {failed} tables")

        self.logger.info("")

        # Save JSON report
        report_file = f"migration_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(report_file, 'w') as f:
            json.dump(self.migration_log, f, indent=2, default=str)
        self.logger.info(f"Report saved to: {report_file}")


def main():
    parser = argparse.ArgumentParser(
        description='Bidirectional PostgreSQL Database Migration',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Dry run (analyze only)
  python migrate_database.py --source synctacles --target energy_insights_nl --dry-run

  # Migrate single table
  python migrate_database.py --source synctacles --target energy_insights_nl --tables alembic_version

  # Migrate all tables
  python migrate_database.py --source synctacles --target energy_insights_nl

  # Reverse direction
        """
    )

    parser.add_argument('--source', required=True, help='Source database name (e.g., synctacles)')
    parser.add_argument('--target', required=True, help='Target database name (e.g., energy_insights_nl)')
    parser.add_argument('--dry-run', action='store_true', help='Analyze only (do not copy data)')
    parser.add_argument('--tables', help='Comma-separated list of tables to migrate (all if omitted)')
    parser.add_argument('--source-host', default='localhost', help='Source host (default: synctacles.com)')
    parser.add_argument('--source-user', default='synctacles', help='Source user (default: synctacles)')
    parser.add_argument('--target-host', default='localhost', help='Target host (default: localhost)')
    parser.add_argument('--target-user', help='Target user (default: same as target database)')

    args = parser.parse_args()

    # Prepare configs
    source_config = {
        'host': args.source_host,
        'port': 5433,
        'database': args.source,
        'user': args.source_user,
        'connect_timeout': 10
    }

    target_config = {
        'host': args.target_host,
        'port': 5432,
        'database': args.target,
        'user': args.target_user or args.target,
        'connect_timeout': 10
    }

    # Execute
    migration = DatabaseMigration(source_config, target_config, dry_run=args.dry_run)

    try:
        migration.connect()

        if args.dry_run:
            migration.analyze()
        else:
            migration.execute(table_filter=args.tables)

    except KeyboardInterrupt:
        print("\nMigration cancelled by user")
        sys.exit(1)
    finally:
        migration.disconnect()


if __name__ == '__main__':
    main()
