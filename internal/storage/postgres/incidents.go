package postgres

import (
	"context"
	"errors"
	"fmt"
	"geoNotifications/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Create inserts a new incident into the database.
func (p *Postgres) Create(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		INSERT INTO incidents (id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at, geog)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography)`

	_, err := p.pool.Exec(ctx, query,
		incident.ID,
		incident.Title,
		incident.Description,
		incident.Latitude,
		incident.Longitude,
		incident.Radius,
		incident.Severity,
		incident.Type,
		incident.IsActive,
		incident.CreatedAt,
		incident.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert incident: %w", err)
	}
	return nil
}

// Update modifies an existing incident.
func (p *Postgres) Update(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		UPDATE incidents
		SET title = $1, description = $2, latitude = $3, longitude = $4, radius = $5,
		    severity = $6, type = $7, is_active = $8, updated_at = $9,
		    geog = ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography
		WHERE id = $10`

	_, err := p.pool.Exec(ctx, query,
		incident.Title,
		incident.Description,
		incident.Latitude,
		incident.Longitude,
		incident.Radius,
		incident.Severity,
		incident.Type,
		incident.IsActive,
		incident.UpdatedAt,
		incident.ID,
	)
	if err != nil {
		return fmt.Errorf("update incident: %w", err)
	}
	return nil
}

// GetAllIncidents returns a paginated list of incidents and the total count.
func (p *Postgres) GetAllIncidents(ctx context.Context, page, limit int, timeout time.Duration) ([]domain.Incident, int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
		FROM incidents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := p.pool.Query(ctx, query, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []domain.Incident
	for rows.Next() {
		var inc domain.Incident
		if err := rows.Scan(
			&inc.ID, &inc.Title, &inc.Description,
			&inc.Latitude, &inc.Longitude, &inc.Radius,
			&inc.Severity, &inc.Type, &inc.IsActive,
			&inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, inc)
	}

	var total int
	if err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM incidents`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count incidents: %w", err)
	}

	return incidents, total, nil
}

// GetByID retrieves a single incident by its ID.
func (p *Postgres) GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
		FROM incidents
		WHERE id = $1`

	var inc domain.Incident
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&inc.ID, &inc.Title, &inc.Description,
		&inc.Latitude, &inc.Longitude, &inc.Radius,
		&inc.Severity, &inc.Type, &inc.IsActive,
		&inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("query incident by id: %w", err)
	}

	return &inc, nil
}

// Delete performs a soft delete by marking the incident inactive.
func (p *Postgres) Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		UPDATE incidents
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1`

	_, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete incident: %w", err)
	}
	return nil
}

// WarmUp loads all active incidents for cache pre-population.
func (p *Postgres) WarmUp(ctx context.Context, timeout time.Duration) ([]*domain.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
		FROM incidents
		WHERE is_active = TRUE`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		var inc domain.Incident
		if err := rows.Scan(
			&inc.ID, &inc.Title, &inc.Description,
			&inc.Latitude, &inc.Longitude, &inc.Radius,
			&inc.Severity, &inc.Type, &inc.IsActive,
			&inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, &inc)
	}

	return incidents, nil
}

// FindNearby returns active incidents whose geofence contains the given point.
// Uses PostGIS ST_DWithin for spatial filtering on the database side.
func (p *Postgres) FindNearby(ctx context.Context, latitude, longitude float64, timeout time.Duration) ([]*domain.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	const query = `
		SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
		FROM incidents
		WHERE is_active = TRUE
		  AND ST_DWithin(geog, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, radius)`

	rows, err := p.pool.Query(ctx, query, longitude, latitude)
	if err != nil {
		return nil, fmt.Errorf("query nearby incidents: %w", err)
	}
	defer rows.Close()

	var incidents []*domain.Incident
	for rows.Next() {
		var inc domain.Incident
		if err := rows.Scan(
			&inc.ID, &inc.Title, &inc.Description,
			&inc.Latitude, &inc.Longitude, &inc.Radius,
			&inc.Severity, &inc.Type, &inc.IsActive,
			&inc.CreatedAt, &inc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan nearby incident: %w", err)
		}
		incidents = append(incidents, &inc)
	}
	return incidents, nil
}
