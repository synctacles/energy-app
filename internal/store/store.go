// Package store provides SQLite-based price caching for the fallback manager.
package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/synctacles/energy-go/internal/models"

	_ "modernc.org/sqlite"
)

// SQLiteCache implements engine.PriceCache using a local SQLite database.
type SQLiteCache struct {
	db *sql.DB
}

// NewSQLiteCache opens (or creates) a SQLite database at configPath/energy_cache.db.
func NewSQLiteCache(configPath string) (*SQLiteCache, error) {
	dbPath := filepath.Join(configPath, "energy_cache.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS prices (
			zone       TEXT    NOT NULL,
			timestamp  TEXT    NOT NULL,
			price_eur  REAL    NOT NULL,
			unit       TEXT    NOT NULL DEFAULT 'kWh',
			source     TEXT    NOT NULL,
			quality    TEXT    NOT NULL DEFAULT 'cached',
			fetched_at TEXT    NOT NULL,
			PRIMARY KEY (zone, timestamp)
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create cache table: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	db.Exec("PRAGMA journal_mode=WAL")

	return &SQLiteCache{db: db}, nil
}

// Close closes the database connection.
func (c *SQLiteCache) Close() error {
	return c.db.Close()
}

// Get retrieves cached prices for a zone and date.
func (c *SQLiteCache) Get(zone string, date time.Time) ([]models.HourlyPrice, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := c.db.Query(`
		SELECT timestamp, price_eur, unit, source, quality, zone
		FROM prices
		WHERE zone = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp
	`, zone, dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("cache query: %w", err)
	}
	defer rows.Close()

	var prices []models.HourlyPrice
	for rows.Next() {
		var tsStr, unitStr, source, quality, z string
		var priceEUR float64
		if err := rows.Scan(&tsStr, &priceEUR, &unitStr, &source, &quality, &z); err != nil {
			continue
		}
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue
		}
		unit := models.UnitKWh
		if unitStr == "MWh" {
			unit = models.UnitMWh
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts,
			PriceEUR:  priceEUR,
			Unit:      unit,
			Source:    source,
			Quality:   "cached",
			Zone:      z,
		})
	}

	return prices, nil
}

// Put stores prices in the cache, upserting on conflict.
func (c *SQLiteCache) Put(zone string, prices []models.HourlyPrice) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO prices (zone, timestamp, price_eur, unit, source, quality, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(zone, timestamp) DO UPDATE SET
			price_eur = excluded.price_eur,
			unit = excluded.unit,
			source = excluded.source,
			quality = excluded.quality,
			fetched_at = excluded.fetched_at
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, p := range prices {
		unitStr := "kWh"
		if p.Unit == models.UnitMWh {
			unitStr = "MWh"
		}
		_, err := stmt.Exec(
			zone,
			p.Timestamp.Format(time.RFC3339),
			p.PriceEUR,
			unitStr,
			p.Source,
			p.Quality,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert price: %w", err)
		}
	}

	return tx.Commit()
}

// Cleanup removes prices older than the given duration.
func (c *SQLiteCache) Cleanup(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)
	result, err := c.db.Exec("DELETE FROM prices WHERE fetched_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
