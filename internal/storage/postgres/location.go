package postgres

import (
	"context"
	"geoNotifications/internal/domain"
	"time"
)

func (p *Postgres) SaveCheckHistory(location *domain.Location, ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	query := `INSERT INTO location_check_history (id, user_id, latitude, longitude, created_at, in_dangerous_area) 
			  VALUES ($1, $2, $3, $4, $5, $6);`
	_, err := p.db.Exec(ctxWithTimeout, query, location.ID, location.UserID, location.Latitude, location.Longitude, location.CreatedAT, location.InDanger)

	return err

}

func (p *Postgres) GetStats(ctx context.Context, timeOut time.Duration) (int, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	statsTime := p.cfg.StatsTime
	var count int
	query := `SELECT COUNT(DISTINCT user_id) AS user_count
			  FROM location_check_history
			  WHERE created_at >= NOW() - ($1 * INTERVAL '1 minute');
				`
	err := p.db.QueryRow(ctxWithTimeout, query, statsTime).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
