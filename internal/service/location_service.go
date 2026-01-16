package service

import (
	"context"
	"errors"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap/zapcore"
)

type Location struct {
	locationDB    LocationDB
	locationCache LocationCache
	locationQueue LocationQueue
	logger        *logger.Service
	config        *config.App
}

type LocationDB interface {
	SaveCheckHistory(location *domain.Location, ctx context.Context, timeOut time.Duration) error
}

type LocationCache interface {
	Check(location *domain.Location, ctx context.Context, timeOut time.Duration) ([]*domain.Incident, error)
}

type LocationQueue interface {
	EnqueueLocationCheck(taskData domain.LocationCheckTask, ctx context.Context, timeOut time.Duration) error
}

func NewLocation(db LocationDB, cache LocationCache, locationQueue LocationQueue, logger *logger.Service, config *config.App) *Location {
	return &Location{
		locationDB:    db,
		locationCache: cache,
		logger:        logger,
		locationQueue: locationQueue,
		config:        config,
	}
}

func (l *Location) Check(latitude, longitude float64, uid string, ctx context.Context) ([]*domain.Incident, error) {

	err := l.isLocationValid(latitude, longitude, uid)
	if err != nil {
		l.logger.Log(zapcore.DebugLevel, "invalid location data: "+err.Error())
		return nil, err
	}
	loc := domain.NewLocation(uuid.MustParse(uid), latitude, longitude)

	incidents, err := l.locationCache.Check(loc, ctx, l.config.DB.TimeOuts.Read)
	if err != nil {
		l.logger.Log(zapcore.WarnLevel, "failed to check location in cache: "+err.Error())
		return nil, err
	}
	if len(incidents) > 0 {
		loc.InDanger = true
	}

	err = l.locationDB.SaveCheckHistory(loc, ctx, l.config.DB.TimeOuts.Write)
	if err != nil {
		l.logger.Log(zapcore.ErrorLevel, "failed to save check history in db: "+err.Error())
	}

	if len(incidents) > 0 {
		if l.locationQueue == nil {
			l.logger.Log(zapcore.ErrorLevel, "location queue is not configured")
			return incidents, errors.New("location queue is not configured")
		}
		var taskData = domain.LocationCheckTask{
			Location:  loc,
			Incidents: incidents,
		}

		err = l.locationQueue.EnqueueLocationCheck(taskData, ctx, l.config.DB.TimeOuts.Write)
		if err != nil {
			l.logger.Log(zapcore.WarnLevel, "failed to enqueue location check: "+err.Error())
		}
	}
	return incidents, nil
}

func (l *Location) isLocationValid(latitude, longitude float64, uid string) error {
	_, err := uuid.Parse(uid)
	if err != nil {
		return err
	}
	if latitude < -90 || latitude > 90 {
		return domain.ErrInvalidLatitude
	}
	if longitude < -180 || longitude > 180 {
		return domain.ErrInvalidLongitude
	}
	return nil
}
