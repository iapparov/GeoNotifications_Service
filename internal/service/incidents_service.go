package service

import (
	"context"
	"errors"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	retry "geoNotifications/internal/rerty"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zapcore"
)

type Incidents struct {
	incidentsDB         IncidentsDB
	incidentsCache      IncidentsCache
	incidentsCacheRetry CacheRetryQueue
	logger              *logger.Service
	config              *config.App
}

type IncidentsDB interface {
	Create(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error
	Update(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error
	GetStats(ctx context.Context, timeOut time.Duration) (int, error)
	GetAllIncidents(ctx context.Context, page, limit int, timeOut time.Duration) ([]domain.Incident, int, error)
	GetByID(ctx context.Context, id uuid.UUID, timeOut time.Duration) (*domain.Incident, error)
	Delete(ctx context.Context, id uuid.UUID, timeOut time.Duration) error
}

type IncidentsCache interface {
	Set(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error
	Delete(ctx context.Context, id uuid.UUID, timeOut time.Duration) error
	GetByID(ctx context.Context, id uuid.UUID, timeOut time.Duration) (*domain.Incident, error)
}

type CacheRetryQueue interface {
	Enqueue(incident *domain.Incident)
}

func NewIncidents(db IncidentsDB, cache IncidentsCache, incidentsCacheRetry CacheRetryQueue, logger *logger.Service, config *config.App) *Incidents {
	return &Incidents{
		incidentsDB:         db,
		incidentsCache:      cache,
		logger:              logger,
		config:              config,
		incidentsCacheRetry: incidentsCacheRetry,
	}
}

func (i *Incidents) Create(title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error) {
	err := i.incidentsValidation(title, description, latitude, longitude, radius, severity, incidentType)
	if err != nil {
		i.logger.Log(zapcore.DebugLevel, err.Error())
		return nil, err
	}
	incident := domain.NewIncident(title, description, latitude, longitude, radius, domain.SeverityLevel(severity), incidentType)

	err = i.incidentsDB.Create(incident, ctx, i.config.DB.TimeOuts.Write)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to create incident in db: "+err.Error())
		return nil, err
	}

	err = retry.DoContext(ctx, retry.Strategy{Attempts: i.config.Retry.MaxAttempts, Delay: i.config.Retry.Delay, Backoff: i.config.Retry.Backoff}, func() error {
		return i.incidentsCache.Set(incident, ctx, i.config.DB.TimeOuts.Write)
	})
	if err != nil {
		i.logger.Log(zapcore.WarnLevel, "failed to create incident in cache: "+err.Error())
		i.incidentsCacheRetry.Enqueue(incident)
	}

	return incident, nil
}

func (i *Incidents) GetStats(ctx context.Context) (int, error) {
	userCount, err := i.incidentsDB.GetStats(ctx, i.config.DB.TimeOuts.Read)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to get stats from db: "+err.Error())
		return 0, err
	}
	return userCount, nil
}

func (i *Incidents) GetAllIncidents(ctx context.Context, page, limit int) ([]domain.Incident, int, error) {
	incidents, count, err := i.incidentsDB.GetAllIncidents(ctx, page, limit, i.config.DB.TimeOuts.Read)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to get all incidents from db: "+err.Error())
		return nil, 0, err
	}
	return incidents, count, nil
}

func (i *Incidents) GetByID(ctx context.Context, id string) (*domain.Incident, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		i.logger.Log(zapcore.DebugLevel, "invalid incident ID: "+err.Error())
		return nil, domain.ErrInvalidID
	}

	incident, err := i.incidentsCache.GetByID(ctx, uid, i.config.DB.TimeOuts.Read)
	if err == nil {
		return incident, nil
	}

	if errors.Is(err, domain.ErrIncidentNotFound) {
		i.logger.Log(zapcore.DebugLevel, "incident not found in cache, fetching from db")
	} else {
		i.logger.Log(zapcore.WarnLevel, "cache error, fallback to db: "+err.Error())
	}

	incident, err = i.incidentsDB.GetByID(ctx, uid, i.config.DB.TimeOuts.Read)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to get incident from db: "+err.Error())
		return nil, err
	}
	_ = i.incidentsCache.Set(incident, ctx, i.config.DB.TimeOuts.Write)

	return incident, nil
}

func (i *Incidents) Update(id, title, description string, latitude, longitude, radius float64, severity, incidentType string, ctx context.Context) (*domain.Incident, error) {
	err := i.incidentsValidation(title, description, latitude, longitude, radius, severity, incidentType)
	if err != nil {
		i.logger.Log(zapcore.DebugLevel, err.Error())
		return nil, err
	}
	incident, err := i.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	incident.Update(title, description, latitude, longitude, radius, domain.SeverityLevel(severity), incidentType)

	err = i.incidentsDB.Update(incident, ctx, i.config.DB.TimeOuts.Write)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to update incident in db: "+err.Error())
		return nil, err
	}

	err = retry.DoContext(ctx, retry.Strategy{Attempts: i.config.Retry.MaxAttempts, Delay: i.config.Retry.Delay, Backoff: i.config.Retry.Backoff}, func() error {
		return i.incidentsCache.Set(incident, ctx, i.config.DB.TimeOuts.Write)
	})
	if err != nil {
		i.logger.Log(zapcore.WarnLevel, "failed to update incident in cache: "+err.Error())
		i.incidentsCacheRetry.Enqueue(incident)
	}

	return incident, nil
}

func (i *Incidents) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		i.logger.Log(zapcore.DebugLevel, "invalid incident ID: "+err.Error())
		return domain.ErrInvalidID
	}

	err = i.incidentsDB.Delete(ctx, uid, i.config.DB.TimeOuts.Write)
	if err != nil {
		i.logger.Log(zapcore.ErrorLevel, "failed to delete incident from db: "+err.Error())
		return err
	}

	err = retry.DoContext(ctx, retry.Strategy{Attempts: i.config.Retry.MaxAttempts, Delay: i.config.Retry.Delay, Backoff: i.config.Retry.Backoff}, func() error {
		return i.incidentsCache.Delete(ctx, uid, i.config.DB.TimeOuts.Write)
	})
	if err != nil {
		i.logger.Log(zapcore.WarnLevel, "failed to delete incident from cache: "+err.Error())
		return err
	}

	return nil
}

func (i *Incidents) incidentsValidation(title, description string, latitude, longitude, radius float64, severity, incidentType string) error {
	if len(title) < i.config.Incident.TitleMinLength || len(title) > i.config.Incident.TitleMaxLength {
		return domain.ErrInvalidTitle
	}
	if len(description) < i.config.Incident.DescriptionMinLength || len(description) > i.config.Incident.DescriptionMaxLength {
		return domain.ErrInvalidDescription
	}
	if latitude < -90 || latitude > 90 {
		return domain.ErrInvalidLatitude
	}
	if longitude < -180 || longitude > 180 {
		return domain.ErrInvalidLongitude
	}
	if radius <= 0 {
		return domain.ErrInvalidRadius
	}
	if severity != string(domain.SeverityLow) && severity != string(domain.SeverityMedium) && severity != string(domain.SeverityHigh) {
		return domain.ErrInvalidSeverity
	}
	if len(incidentType) == 0 {
		return domain.ErrInvalidType
	}
	return nil
}
