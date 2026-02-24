// Package store — PostgreSQL price store for the central energy server.
// Implements the engine.PriceCache interface, same as SQLiteCache but with PostgreSQL.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/synctacles/energy-app/pkg/models"
)

// PostgresStore implements engine.PriceCache using PostgreSQL.
// Used by the central energy-server to store prices for all 30 EU zones.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a PostgreSQL connection and ensures the schema exists.
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := migratePostgres(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate postgres: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// migratePostgres creates the prices table if it doesn't exist.
func migratePostgres(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS prices (
			zone        TEXT        NOT NULL,
			timestamp   TIMESTAMPTZ NOT NULL,
			price_eur   NUMERIC(10,6) NOT NULL,
			unit        TEXT        NOT NULL DEFAULT 'kWh',
			source      TEXT        NOT NULL,
			quality     TEXT        NOT NULL DEFAULT 'live',
			is_consumer BOOLEAN     NOT NULL DEFAULT false,
			fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (zone, timestamp)
		);

		CREATE INDEX IF NOT EXISTS idx_prices_zone_date
			ON prices (zone, timestamp DESC);
	`)
	return err
}

// Close closes the database connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// Get retrieves prices for a zone and date.
func (s *PostgresStore) Get(zone string, date time.Time) ([]models.HourlyPrice, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := s.db.Query(`
		SELECT timestamp, price_eur, unit, source, quality, zone, is_consumer
		FROM prices
		WHERE zone = $1 AND timestamp >= $2 AND timestamp < $3
		ORDER BY timestamp
	`, zone, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("query prices: %w", err)
	}
	defer rows.Close()

	var prices []models.HourlyPrice
	for rows.Next() {
		var ts time.Time
		var priceEUR float64
		var unitStr, source, quality, z string
		var isConsumer bool
		if err := rows.Scan(&ts, &priceEUR, &unitStr, &source, &quality, &z, &isConsumer); err != nil {
			continue
		}
		unit := models.UnitKWh
		if unitStr == "MWh" || unitStr == "EUR/MWh" {
			unit = models.UnitMWh
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   priceEUR,
			Unit:       unit,
			Source:     source,
			Quality:    quality,
			Zone:       z,
			IsConsumer: isConsumer,
		})
	}
	return prices, nil
}

// Put stores prices in the database, upserting on conflict.
func (s *PostgresStore) Put(zone string, prices []models.HourlyPrice) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO prices (zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (zone, timestamp) DO UPDATE SET
			price_eur = EXCLUDED.price_eur,
			unit = EXCLUDED.unit,
			source = EXCLUDED.source,
			quality = EXCLUDED.quality,
			is_consumer = EXCLUDED.is_consumer,
			fetched_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, p := range prices {
		unitStr := "kWh"
		if p.Unit == models.UnitMWh {
			unitStr = "MWh"
		}
		if _, err := stmt.Exec(zone, p.Timestamp.UTC(), p.PriceEUR, unitStr, p.Source, p.Quality, p.IsConsumer); err != nil {
			return fmt.Errorf("insert price: %w", err)
		}
	}

	return tx.Commit()
}

// GetAllZonesLatest returns the most recent prices for all zones that have data for the given date.
// Used by the API to serve cached responses.
func (s *PostgresStore) GetAllZonesLatest(date time.Time) (map[string][]models.HourlyPrice, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := s.db.Query(`
		SELECT timestamp, price_eur, unit, source, quality, zone, is_consumer
		FROM prices
		WHERE timestamp >= $1 AND timestamp < $2
		ORDER BY zone, timestamp
	`, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("query all zones: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]models.HourlyPrice)
	for rows.Next() {
		var ts time.Time
		var priceEUR float64
		var unitStr, source, quality, zone string
		var isConsumer bool
		if err := rows.Scan(&ts, &priceEUR, &unitStr, &source, &quality, &zone, &isConsumer); err != nil {
			continue
		}
		unit := models.UnitKWh
		if unitStr == "MWh" || unitStr == "EUR/MWh" {
			unit = models.UnitMWh
		}
		result[zone] = append(result[zone], models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   priceEUR,
			Unit:       unit,
			Source:     source,
			Quality:    quality,
			Zone:       zone,
			IsConsumer: isConsumer,
		})
	}
	return result, nil
}

// Cleanup removes prices older than the given duration.
func (s *PostgresStore) Cleanup(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-maxAge)
	result, err := s.db.Exec("DELETE FROM prices WHERE fetched_at < $1", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
