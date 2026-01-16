package postgres

import (
	"context"
	"database/sql"
	"errors"
	"geoNotifications/internal/domain"
	"time"

	"github.com/google/uuid"
)

func (p *Postgres) Create(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	query := `
		INSERT INTO incidents (id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
		`

	_, err := p.db.Exec(ctxWithTimeout,
		query,
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
		return err
	}
	return nil
}

func (p *Postgres) Update(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	query := `
		UPDATE incidents
		SET title = $1, description = $2, latitude = $3, longitude = $4, radius = $5, severity = $6, type = $7, is_active = $8, updated_at = $9
		WHERE id = $10;
		`

	_, err := p.db.Exec(ctxWithTimeout,
		query,
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
		return err
	}
	return nil
}

func (p *Postgres) GetAllIncidents(ctx context.Context, page, limit int, timeOut time.Duration) ([]domain.Incident, int, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	var incidents []domain.Incident
	var totalCount int

	query := `SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
			  FROM incidents
			  ORDER BY created_at DESC
			  LIMIT $1 OFFSET $2;`

	rows, err := p.db.Query(ctxWithTimeout, query, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var incident domain.Incident
		if err := rows.Scan(
			&incident.ID,
			&incident.Title,
			&incident.Description,
			&incident.Latitude,
			&incident.Longitude,
			&incident.Radius,
			&incident.Severity,
			&incident.Type,
			&incident.IsActive,
			&incident.CreatedAt,
			&incident.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		incidents = append(incidents, incident)
	}

	countQuery := `SELECT COUNT(*) FROM incidents;`
	err = p.db.QueryRow(ctxWithTimeout, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	return incidents, totalCount, nil
}

func (p *Postgres) GetByID(ctx context.Context, id uuid.UUID, timeOut time.Duration) (*domain.Incident, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	var incident domain.Incident

	query := `SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
			  FROM incidents
			  WHERE id = $1;`

	err := p.db.QueryRow(ctxWithTimeout, query, id).Scan(
		&incident.ID,
		&incident.Title,
		&incident.Description,
		&incident.Latitude,
		&incident.Longitude,
		&incident.Radius,
		&incident.Severity,
		&incident.Type,
		&incident.IsActive,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, err
	}

	return &incident, nil
}
func (p *Postgres) Delete(ctx context.Context, id uuid.UUID, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	query := `
		UPDATE incidents
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1;
		`
	_, err := p.db.Exec(ctxWithTimeout, query, id)
	if err != nil {
		return err
	}
	return nil
}

func (p *Postgres) WarmUp(ctx context.Context, timeOut time.Duration) ([]*domain.Incident, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	var incidents []*domain.Incident

	query := `SELECT id, title, description, latitude, longitude, radius, severity, type, is_active, created_at, updated_at
			  FROM incidents
			  WHERE is_active = TRUE;`

	rows, err := p.db.Query(ctxWithTimeout, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var incident domain.Incident
		if err := rows.Scan(
			&incident.ID,
			&incident.Title,
			&incident.Description,
			&incident.Latitude,
			&incident.Longitude,
			&incident.Radius,
			&incident.Severity,
			&incident.Type,
			&incident.IsActive,
			&incident.CreatedAt,
			&incident.UpdatedAt,
		); err != nil {
			return nil, err
		}
		incidents = append(incidents, &incident)
	}

	return incidents, nil
}
