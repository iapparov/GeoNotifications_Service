package service

import (
	"context"
	"errors"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"geoNotifications/internal/retry"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zapcore"
)

// IncidentsDB defines the database operations for incidents.
type IncidentsDB interface {
	Create(ctx context.Context, incident *domain.Incident, timeout time.Duration) error
	Update(ctx context.Context, incident *domain.Incident, timeout time.Duration) error
	GetStats(ctx context.Context, timeout time.Duration) (int, error)
	GetAllIncidents(ctx context.Context, page, limit int, timeout time.Duration) ([]domain.Incident, int, error)
	GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error)
	Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error
}

// IncidentsCache defines the cache operations for incidents.
type IncidentsCache interface {
	Set(ctx context.Context, incident *domain.Incident, timeout time.Duration) error
	Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error
	GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error)
}

// CacheRetryQueue enqueues failed cache writes for retry.
type CacheRetryQueue interface {
	Enqueue(incident *domain.Incident)
}

// Incidents handles incident business logic.
type Incidents struct {
	db         IncidentsDB
	cache      IncidentsCache
	retryQueue CacheRetryQueue
	log        *logger.Service
	cfg        *config.App
}

// NewIncidents creates an Incidents service.
func NewIncidents(db IncidentsDB, cache IncidentsCache, retryQueue CacheRetryQueue, log *logger.Service, cfg *config.App) *Incidents {
	return &Incidents{
		db:         db,
		cache:      cache,
		retryQueue: retryQueue,
		log:        log,
		cfg:        cfg,
	}
}

// Create validates input, persists a new incident, and writes it to cache.
func (s *Incidents) Create(ctx context.Context, title, description string, lat, lon, radius float64, severity, incidentType string) (*domain.Incident, error) {
	if err := s.validate(title, description, lat, lon, radius, severity, incidentType); err != nil {
		s.log.Log(zapcore.DebugLevel, err.Error())
		return nil, err
	}

	inc := domain.NewIncident(title, description, lat, lon, radius, domain.SeverityLevel(severity), incidentType)

	if err := s.db.Create(ctx, inc, s.cfg.DB.TimeOuts.Write); err != nil {
		s.log.Log(zapcore.ErrorLevel, "db create incident: "+err.Error())
		return nil, err
	}

	if err := retry.Do(ctx, s.retryStrategy(), func() error {
		return s.cache.Set(ctx, inc, s.cfg.DB.TimeOuts.Write)
	}); err != nil {
		s.log.Log(zapcore.WarnLevel, "cache set incident: "+err.Error())
		s.retryQueue.Enqueue(inc)
	}

	return inc, nil
}

// GetStats returns the number of unique users who checked location recently.
func (s *Incidents) GetStats(ctx context.Context) (int, error) {
	count, err := s.db.GetStats(ctx, s.cfg.DB.TimeOuts.Read)
	if err != nil {
		s.log.Log(zapcore.ErrorLevel, "db get stats: "+err.Error())
		return 0, err
	}
	return count, nil
}

// GetAllIncidents returns a paginated list of incidents.
func (s *Incidents) GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error) {
	incidents, count, err := s.db.GetAllIncidents(ctx, page, limit, s.cfg.DB.TimeOuts.Read)
	if err != nil {
		s.log.Log(zapcore.ErrorLevel, "db get all incidents: "+err.Error())
		return nil, 0, err
	}
	return incidents, count, nil
}

// GetByID looks up an incident in cache first, then falls back to the database.
func (s *Incidents) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		s.log.Log(zapcore.DebugLevel, "invalid incident ID: "+err.Error())
		return nil, domain.ErrInvalidID
	}

	inc, err := s.cache.GetByID(ctx, uid, s.cfg.DB.TimeOuts.Read)
	if err == nil {
		return inc, nil
	}

	if errors.Is(err, domain.ErrIncidentNotFound) {
		s.log.Log(zapcore.DebugLevel, "incident not in cache, querying db")
	} else {
		s.log.Log(zapcore.WarnLevel, "cache error, fallback to db: "+err.Error())
	}

	inc, err = s.db.GetByID(ctx, uid, s.cfg.DB.TimeOuts.Read)
	if err != nil {
		s.log.Log(zapcore.ErrorLevel, "db get incident: "+err.Error())
		return nil, err
	}

	_ = s.cache.Set(ctx, inc, s.cfg.DB.TimeOuts.Write)
	return inc, nil
}

// Update validates input, updates the incident in the database and cache.
func (s *Incidents) Update(ctx context.Context, id, title, description string, lat, lon, radius float64, severity, incidentType string) (*domain.Incident, error) {
	if err := s.validate(title, description, lat, lon, radius, severity, incidentType); err != nil {
		s.log.Log(zapcore.DebugLevel, err.Error())
		return nil, err
	}

	inc, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	inc.Update(title, description, lat, lon, radius, domain.SeverityLevel(severity), incidentType)

	if err := s.db.Update(ctx, inc, s.cfg.DB.TimeOuts.Write); err != nil {
		s.log.Log(zapcore.ErrorLevel, "db update incident: "+err.Error())
		return nil, err
	}

	if err := retry.Do(ctx, s.retryStrategy(), func() error {
		return s.cache.Set(ctx, inc, s.cfg.DB.TimeOuts.Write)
	}); err != nil {
		s.log.Log(zapcore.WarnLevel, "cache update incident: "+err.Error())
		s.retryQueue.Enqueue(inc)
	}

	return inc, nil
}

// Delete soft-deletes the incident from the database and removes it from cache.
func (s *Incidents) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		s.log.Log(zapcore.DebugLevel, "invalid incident ID: "+err.Error())
		return domain.ErrInvalidID
	}

	if err := s.db.Delete(ctx, uid, s.cfg.DB.TimeOuts.Write); err != nil {
		s.log.Log(zapcore.ErrorLevel, "db delete incident: "+err.Error())
		return err
	}

	if err := retry.Do(ctx, s.retryStrategy(), func() error {
		return s.cache.Delete(ctx, uid, s.cfg.DB.TimeOuts.Write)
	}); err != nil {
		s.log.Log(zapcore.WarnLevel, "cache delete incident: "+err.Error())
		return err
	}

	return nil
}

func (s *Incidents) retryStrategy() retry.Strategy {
	return retry.Strategy{
		Attempts: s.cfg.Retry.MaxAttempts,
		Delay:    s.cfg.Retry.Delay,
		Backoff:  s.cfg.Retry.Backoff,
	}
}

func (s *Incidents) validate(title, description string, lat, lon, radius float64, severity, incidentType string) error {
	if len(title) < s.cfg.Incident.TitleMinLength || len(title) > s.cfg.Incident.TitleMaxLength {
		return domain.ErrInvalidTitle
	}
	if len(description) < s.cfg.Incident.DescriptionMinLength || len(description) > s.cfg.Incident.DescriptionMaxLength {
		return domain.ErrInvalidDescription
	}
	if lat < -90 || lat > 90 {
		return domain.ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return domain.ErrInvalidLongitude
	}
	if radius <= 0 {
		return domain.ErrInvalidRadius
	}
	switch domain.SeverityLevel(severity) {
	case domain.SeverityLow, domain.SeverityMedium, domain.SeverityHigh:
	default:
		return domain.ErrInvalidSeverity
	}
	if incidentType == "" {
		return domain.ErrInvalidType
	}
	return nil
}
