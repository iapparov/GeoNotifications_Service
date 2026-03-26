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

// LocationDB defines database operations for location checks.
type LocationDB interface {
	SaveCheckHistory(ctx context.Context, loc *domain.Location, timeout time.Duration) error
	FindNearby(ctx context.Context, lat, lon float64, timeout time.Duration) ([]*domain.Incident, error)
}

// LocationQueue defines the interface for enqueuing location check tasks.
type LocationQueue interface {
	EnqueueLocationCheck(ctx context.Context, task domain.LocationCheckTask, timeout time.Duration) error
}

// Location handles location check business logic.
type Location struct {
	db    LocationDB
	queue LocationQueue
	log   *logger.Service
	cfg   *config.App
}

// NewLocation creates a Location service.
func NewLocation(db LocationDB, queue LocationQueue, log *logger.Service, cfg *config.App) *Location {
	return &Location{
		db:    db,
		queue: queue,
		log:   log,
		cfg:   cfg,
	}
}

// Check verifies whether the user is within any active incident geofence.
// If the user is in danger, the check is enqueued for webhook notification.
func (s *Location) Check(ctx context.Context, lat, lon float64, uid string) ([]*domain.Incident, error) {
	if err := s.validateInput(lat, lon, uid); err != nil {
		s.log.Log(zapcore.DebugLevel, "invalid location data: "+err.Error())
		return nil, err
	}

	loc := domain.NewLocation(uuid.MustParse(uid), lat, lon)

	incidents, err := s.db.FindNearby(ctx, lat, lon, s.cfg.DB.TimeOuts.Read)
	if err != nil {
		s.log.Log(zapcore.WarnLevel, "find nearby incidents: "+err.Error())
		return nil, err
	}

	if len(incidents) > 0 {
		loc.InDanger = true
	}

	if err := s.db.SaveCheckHistory(ctx, loc, s.cfg.DB.TimeOuts.Write); err != nil {
		s.log.Log(zapcore.ErrorLevel, "save check history: "+err.Error())
	}

	if len(incidents) > 0 {
		if s.queue == nil {
			s.log.Log(zapcore.ErrorLevel, "location queue is not configured")
			return incidents, errors.New("location queue is not configured")
		}

		task := domain.LocationCheckTask{
			Location:  loc,
			Incidents: incidents,
		}
		if err := s.queue.EnqueueLocationCheck(ctx, task, s.cfg.DB.TimeOuts.Write); err != nil {
			s.log.Log(zapcore.WarnLevel, "enqueue location check: "+err.Error())
		}
	}

	return incidents, nil
}

func (s *Location) validateInput(lat, lon float64, uid string) error {
	if _, err := uuid.Parse(uid); err != nil {
		return domain.ErrInvalidID
	}
	if lat < -90 || lat > 90 {
		return domain.ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return domain.ErrInvalidLongitude
	}
	return nil
}
