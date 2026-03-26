package postgres

import (
	"context"
	"fmt"
	"geoNotifications/internal/domain"
	"time"
)

// SaveCheckHistory persists a location check record.
func (p *Postgres) SaveCheckHistory(ctx context.Context, loc *domain.Location, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		INSERT INTO location_check_history (id, user_id, latitude, longitude, created_at, in_dangerous_area)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := p.pool.Exec(ctx, query,
		loc.ID, loc.UserID, loc.Latitude, loc.Longitude, loc.CreatedAt, loc.InDanger,
	)
	if err != nil {
		return fmt.Errorf("save check history: %w", err)
	}
	return nil
}

// GetStats returns the number of distinct users who checked their location
// within the configured time window.
func (p *Postgres) GetStats(ctx context.Context, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		SELECT COUNT(DISTINCT user_id)
		FROM location_check_history
		WHERE created_at >= NOW() - ($1 * INTERVAL '1 minute')`

	var count int
	if err := p.pool.QueryRow(ctx, query, p.cfg.StatsTime).Scan(&count); err != nil {
		return 0, fmt.Errorf("get stats: %w", err)
	}
	return count, nil
}
