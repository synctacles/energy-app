// Package store provides SQLite-based price caching for the fallback manager.
package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/synctacles/energy-app/pkg/models"

	_ "modernc.org/sqlite"
)

// SQLiteCache implements engine.PriceCache and engine.SmartPriceCache
// using a local SQLite database.
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

	// Migrations (idempotent — errors ignored for existing columns)
	_, _ = db.Exec("ALTER TABLE prices ADD COLUMN original_tier INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE prices ADD COLUMN is_consumer INTEGER NOT NULL DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE prices ADD COLUMN wholesale_kwh REAL NOT NULL DEFAULT 0")

	// Data fix: prices stored as kWh by synctacles/enever are consumer prices
	// but got is_consumer=0 from the DEFAULT when the column was first added.
	// Idempotent — no-op once all rows are already corrected.
	_, _ = db.Exec("UPDATE prices SET is_consumer = 1 WHERE unit = 'kWh' AND is_consumer = 0 AND source IN ('synctacles', 'enever')")

	// Enable WAL mode for better concurrent read performance
	_, _ = db.Exec("PRAGMA journal_mode=WAL")

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
		SELECT timestamp, price_eur, unit, source, quality, zone, is_consumer, wholesale_kwh
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
		var priceEUR, wholesaleKWh float64
		var isConsumer int
		if err := rows.Scan(&tsStr, &priceEUR, &unitStr, &source, &quality, &z, &isConsumer, &wholesaleKWh); err != nil {
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
			Timestamp:    ts,
			PriceEUR:     priceEUR,
			WholesaleKWh: wholesaleKWh,
			Unit:         unit,
			Source:       source,
			Quality:      quality,
			Zone:         z,
			IsConsumer:   isConsumer == 1,
		})
	}

	return prices, nil
}

// GetWithMeta retrieves cached prices with provenance metadata.
// Returns nil (not error) when no rows are found.
func (c *SQLiteCache) GetWithMeta(zone string, date time.Time) (*models.CacheEntry, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	rows, err := c.db.Query(`
		SELECT timestamp, price_eur, unit, source, quality, zone, original_tier, fetched_at, is_consumer, wholesale_kwh
		FROM prices
		WHERE zone = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp
	`, zone, dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("cache query: %w", err)
	}
	defer rows.Close()

	var prices []models.HourlyPrice
	var tier int
	var fetchedAt time.Time
	for rows.Next() {
		var tsStr, unitStr, source, quality, z, fetchedAtStr string
		var priceEUR, wholesaleKWh float64
		var rowTier, isConsumer int
		if err := rows.Scan(&tsStr, &priceEUR, &unitStr, &source, &quality, &z, &rowTier, &fetchedAtStr, &isConsumer, &wholesaleKWh); err != nil {
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
			Timestamp:    ts,
			PriceEUR:     priceEUR,
			WholesaleKWh: wholesaleKWh,
			Unit:         unit,
			Source:       source,
			Quality:      quality,
			Zone:         z,
			IsConsumer:   isConsumer == 1,
		})
		// Use tier and fetched_at from last row (all rows in a batch share the same values)
		tier = rowTier
		if fa, err := time.Parse(time.RFC3339, fetchedAtStr); err == nil {
			fetchedAt = fa
		}
	}

	if len(prices) == 0 {
		return nil, nil
	}

	return &models.CacheEntry{
		Prices:       prices,
		OriginalTier: tier,
		FetchedAt:    fetchedAt,
	}, nil
}

// Put stores prices in the cache with tier 0 (unknown provenance).
func (c *SQLiteCache) Put(zone string, prices []models.HourlyPrice) error {
	return c.PutWithTier(zone, prices, 0)
}

// PutWithTier stores prices in the cache with the original source tier.
func (c *SQLiteCache) PutWithTier(zone string, prices []models.HourlyPrice, tier int) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO prices (zone, timestamp, price_eur, unit, source, quality, fetched_at, original_tier, is_consumer, wholesale_kwh)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(zone, timestamp) DO UPDATE SET
			price_eur = excluded.price_eur,
			unit = excluded.unit,
			source = excluded.source,
			quality = excluded.quality,
			fetched_at = excluded.fetched_at,
			original_tier = excluded.original_tier,
			is_consumer = excluded.is_consumer,
			wholesale_kwh = excluded.wholesale_kwh
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
		isConsumer := 0
		if p.IsConsumer {
			isConsumer = 1
		}
		_, err := stmt.Exec(
			zone,
			p.Timestamp.Format(time.RFC3339),
			p.PriceEUR,
			unitStr,
			p.Source,
			p.Quality,
			now,
			tier,
			isConsumer,
			p.WholesaleKWh,
		)
		if err != nil {
			return fmt.Errorf("insert price: %w", err)
		}
	}

	return tx.Commit()
}

// CacheRow represents a single cached price row with all metadata.
type CacheRow struct {
	Timestamp    string  `json:"timestamp"`
	PriceEUR     float64 `json:"price_eur"`
	WholesaleKWh float64 `json:"wholesale_kwh"`
	Unit         string  `json:"unit"`
	Source       string  `json:"source"`
	Quality      string  `json:"quality"`
	FetchedAt    string  `json:"fetched_at"`
	Tier         int     `json:"tier"`
	IsConsumer   bool    `json:"is_consumer"`
}

// GetAllForZone returns all cached price rows for a zone covering today + tomorrow.
func (c *SQLiteCache) GetAllForZone(zone string, today time.Time) ([]CacheRow, error) {
	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(48 * time.Hour)

	rows, err := c.db.Query(`
		SELECT timestamp, price_eur, wholesale_kwh, unit, source, quality, fetched_at, original_tier, is_consumer
		FROM prices
		WHERE zone = ? AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp
	`, zone, dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("cache query all: %w", err)
	}
	defer rows.Close()

	var result []CacheRow
	for rows.Next() {
		var r CacheRow
		var isConsumer int
		if err := rows.Scan(&r.Timestamp, &r.PriceEUR, &r.WholesaleKWh, &r.Unit, &r.Source, &r.Quality, &r.FetchedAt, &r.Tier, &isConsumer); err != nil {
			continue
		}
		r.IsConsumer = isConsumer == 1
		result = append(result, r)
	}
	return result, nil
}

// ClearZone deletes all cached prices for a zone. Returns rows deleted.
func (c *SQLiteCache) ClearZone(zone string) (int64, error) {
	result, err := c.db.Exec("DELETE FROM prices WHERE zone = ?", zone)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Cleanup removes prices whose hour timestamp is older than the given duration.
// Uses the price timestamp (not fetched_at) because day-ahead prices are immutable —
// a price for yesterday 15:00 is never useful regardless of when it was fetched.
func (c *SQLiteCache) Cleanup(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339)
	result, err := c.db.Exec("DELETE FROM prices WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
