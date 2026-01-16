package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geoNotifications/internal/domain"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

func (r *Redis) Set(incident *domain.Incident, ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	data, err := json.Marshal(incident)
	if err != nil {
		return fmt.Errorf("marshal incident: %w", err)
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctxWithTimeout, incident.ID.String(), data, 0)
	pipe.SAdd(ctxWithTimeout, "active_incident_ids", incident.ID.String())

	_, err = pipe.Exec(ctxWithTimeout)

	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (r *Redis) Delete(ctx context.Context, id uuid.UUID, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	pipe := r.client.TxPipeline()
	pipe.Del(ctxWithTimeout, id.String())
	pipe.SRem(ctxWithTimeout, "active_incident_ids", id.String())
	_, err := pipe.Exec(ctxWithTimeout)

	if err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}

	return nil
}
func (r *Redis) GetByID(ctx context.Context, id uuid.UUID, timeOut time.Duration) (*domain.Incident, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	data, err := r.client.Get(ctxWithTimeout, id.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var incident domain.Incident
	err = json.Unmarshal([]byte(data), &incident)
	if err != nil {
		return nil, fmt.Errorf("unmarshal incident: %w", err)
	}

	return &incident, nil
}
func (r *Redis) Check(location *domain.Location, ctx context.Context, timeOut time.Duration) ([]*domain.Incident, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	ids, err := r.client.SMembers(ctxWithTimeout, "active_incident_ids").Result()
	if err != nil {
		return nil, fmt.Errorf("redis SMEMBERS: %w", err)
	}

	if len(ids) == 0 {
		return []*domain.Incident{}, nil
	}

	data, err := r.client.MGet(ctxWithTimeout, ids...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis MGET: %w", err)
	}

	incidentsInArea := make([]*domain.Incident, 0, len(data))
	for _, d := range data {
		str, ok := d.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected redis MGET type %T", d)
		}
		var incident domain.Incident
		err := json.Unmarshal([]byte(str), &incident)
		if err != nil {
			return nil, fmt.Errorf("unmarshal incident: %w", err)
		}
		if incident.IsLocationInIncidentArea(location.Latitude, location.Longitude) {
			incidentsInArea = append(incidentsInArea, &incident)
		}
	}
	return incidentsInArea, nil
}

func (r *Redis) WarmUp(ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()

	incidents, err := r.db.WarmUp(ctxWithTimeout, timeOut)

	if err != nil {
		return fmt.Errorf("redis warm up: %w", err)
	}

	pipe := r.client.TxPipeline()

	pipe.Del(ctxWithTimeout, "active_incident_ids") // очистка старого индекса

	for _, inc := range incidents {
		data, _ := json.Marshal(inc)
		pipe.Set(ctxWithTimeout, inc.ID.String(), data, 0)
		pipe.SAdd(ctxWithTimeout, "active_incident_ids", inc.ID.String())
	}

	_, err = pipe.Exec(ctxWithTimeout)
	if err != nil {
		return fmt.Errorf("redis warm up: %w", err)
	}
	return nil
}
