package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/synctacles/energy-go/internal/api/models"
)

// Repository handles all database operations for the Energy API.
type Repository struct {
	db *pgxpool.Pool
}

// New creates a new Repository with a database connection pool.
func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetPrices fetches energy prices for a given zone and time range.
func (r *Repository) GetPrices(ctx context.Context, zone string, start, end time.Time) ([]models.Price, error) {
	query := `
		SELECT zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at
		FROM prices
		WHERE zone = $1
		  AND timestamp >= $2
		  AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Query(ctx, query, zone, start, end)
	if err != nil {
		return nil, fmt.Errorf("query prices: %w", err)
	}
	defer rows.Close()

	var prices []models.Price
	for rows.Next() {
		var p models.Price
		err := rows.Scan(
			&p.Zone,
			&p.Timestamp,
			&p.PriceEUR,
			&p.Unit,
			&p.Source,
			&p.Quality,
			&p.IsConsumer,
			&p.FetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		prices = append(prices, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prices: %w", err)
	}

	return prices, nil
}

// GetCurrentPrice fetches the most recent price for a zone.
func (r *Repository) GetCurrentPrice(ctx context.Context, zone string) (*models.Price, error) {
	query := `
		SELECT zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at
		FROM prices
		WHERE zone = $1
		  AND timestamp <= NOW()
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var p models.Price
	err := r.db.QueryRow(ctx, query, zone).Scan(
		&p.Zone,
		&p.Timestamp,
		&p.PriceEUR,
		&p.Unit,
		&p.Source,
		&p.Quality,
		&p.IsConsumer,
		&p.FetchedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query current price: %w", err)
	}

	return &p, nil
}

// GetTodayStats calculates statistics for today's prices in a zone.
func (r *Repository) GetTodayStats(ctx context.Context, zone string) (avg, min, max float64, err error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT
			COALESCE(AVG(price_eur), 0) as avg_price,
			COALESCE(MIN(price_eur), 0) as min_price,
			COALESCE(MAX(price_eur), 0) as max_price
		FROM prices
		WHERE zone = $1
		  AND timestamp >= $2
		  AND timestamp < $3
	`

	err = r.db.QueryRow(ctx, query, zone, startOfDay, endOfDay).Scan(&avg, &min, &max)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("query today stats: %w", err)
	}

	return avg, min, max, nil
}

// GetCheapestToday finds the cheapest hour today for a zone.
func (r *Repository) GetCheapestToday(ctx context.Context, zone string) (*models.Price, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at
		FROM prices
		WHERE zone = $1
		  AND timestamp >= $2
		  AND timestamp < $3
		ORDER BY price_eur ASC
		LIMIT 1
	`

	var p models.Price
	err := r.db.QueryRow(ctx, query, zone, startOfDay, endOfDay).Scan(
		&p.Zone,
		&p.Timestamp,
		&p.PriceEUR,
		&p.Unit,
		&p.Source,
		&p.Quality,
		&p.IsConsumer,
		&p.FetchedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query cheapest today: %w", err)
	}

	return &p, nil
}

// GetMostExpensiveToday finds the most expensive hour today for a zone.
func (r *Repository) GetMostExpensiveToday(ctx context.Context, zone string) (*models.Price, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at
		FROM prices
		WHERE zone = $1
		  AND timestamp >= $2
		  AND timestamp < $3
		ORDER BY price_eur DESC
		LIMIT 1
	`

	var p models.Price
	err := r.db.QueryRow(ctx, query, zone, startOfDay, endOfDay).Scan(
		&p.Zone,
		&p.Timestamp,
		&p.PriceEUR,
		&p.Unit,
		&p.Source,
		&p.Quality,
		&p.IsConsumer,
		&p.FetchedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query most expensive today: %w", err)
	}

	return &p, nil
}

// GetNextHourPrice fetches the price for the next hour.
func (r *Repository) GetNextHourPrice(ctx context.Context, zone string) (*models.Price, error) {
	now := time.Now()
	nextHour := now.Add(1 * time.Hour).Truncate(1 * time.Hour)

	query := `
		SELECT zone, timestamp, price_eur, unit, source, quality, is_consumer, fetched_at
		FROM prices
		WHERE zone = $1
		  AND timestamp = $2
		LIMIT 1
	`

	var p models.Price
	err := r.db.QueryRow(ctx, query, zone, nextHour).Scan(
		&p.Zone,
		&p.Timestamp,
		&p.PriceEUR,
		&p.Unit,
		&p.Source,
		&p.Quality,
		&p.IsConsumer,
		&p.FetchedAt,
	)
	if err != nil {
		// Next hour price may not be available yet - not an error
		return nil, nil
	}

	return &p, nil
}

// GetBestWindow calculates the cheapest consecutive window of N hours.
func (r *Repository) GetBestWindow(ctx context.Context, zone string, windowHours int, startTime, endTime time.Time) (*models.BestWindowResponse, error) {
	// Fetch all prices in the range
	prices, err := r.GetPrices(ctx, zone, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get prices for best window: %w", err)
	}

	if len(prices) < windowHours {
		return nil, fmt.Errorf("not enough prices (%d) for window size %d", len(prices), windowHours)
	}

	// Calculate sliding window averages
	var bestStart int
	var bestAvg = 999999.0
	var overallAvg = 0.0

	for i := 0; i <= len(prices)-windowHours; i++ {
		windowSum := 0.0
		for j := 0; j < windowHours; j++ {
			windowSum += prices[i+j].PriceEUR
		}
		windowAvg := windowSum / float64(windowHours)

		if windowAvg < bestAvg {
			bestAvg = windowAvg
			bestStart = i
		}
	}

	// Calculate overall average for savings comparison
	totalSum := 0.0
	for _, p := range prices {
		totalSum += p.PriceEUR
	}
	overallAvg = totalSum / float64(len(prices))

	return &models.BestWindowResponse{
		Zone:       zone,
		WindowSize: windowHours,
		StartTime:  prices[bestStart].Timestamp,
		EndTime:    prices[bestStart+windowHours-1].Timestamp,
		AvgPrice:   bestAvg,
		Savings:    overallAvg - bestAvg,
	}, nil
}

// GetTomorrowPrices fetches all prices for tomorrow in a zone.
func (r *Repository) GetTomorrowPrices(ctx context.Context, zone string) ([]models.Price, error) {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	startOfTomorrow := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
	endOfTomorrow := startOfTomorrow.Add(24 * time.Hour)

	return r.GetPrices(ctx, zone, startOfTomorrow, endOfTomorrow)
}
