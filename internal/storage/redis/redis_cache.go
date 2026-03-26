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

const activeIDsKey = "active_incident_ids"

// Set stores an incident in Redis and adds its ID to the active set.
func (r *Redis) Set(ctx context.Context, incident *domain.Incident, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := json.Marshal(incident)
	if err != nil {
		return fmt.Errorf("marshal incident: %w", err)
	}

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, incident.ID.String(), data, 0)
	pipe.SAdd(ctx, activeIDsKey, incident.ID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis set incident: %w", err)
	}
	return nil
}

// Delete removes an incident from Redis and the active set.
func (r *Redis) Delete(ctx context.Context, id uuid.UUID, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, id.String())
	pipe.SRem(ctx, activeIDsKey, id.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis delete incident: %w", err)
	}
	return nil
}

// GetByID retrieves a single incident from Redis.
func (r *Redis) GetByID(ctx context.Context, id uuid.UUID, timeout time.Duration) (*domain.Incident, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := r.client.Get(ctx, id.String()).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var inc domain.Incident
	if err := json.Unmarshal([]byte(data), &inc); err != nil {
		return nil, fmt.Errorf("unmarshal incident: %w", err)
	}
	return &inc, nil
}

// WarmUp loads all active incidents from the database into Redis.
func (r *Redis) WarmUp(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	incidents, err := r.db.WarmUp(ctx, timeout)
	if err != nil {
		return fmt.Errorf("warmup load: %w", err)
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, activeIDsKey)

	for _, inc := range incidents {
		data, err := json.Marshal(inc)
		if err != nil {
			return fmt.Errorf("marshal incident %s: %w", inc.ID, err)
		}
		pipe.Set(ctx, inc.ID.String(), data, 0)
		pipe.SAdd(ctx, activeIDsKey, inc.ID.String())
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("warmup exec: %w", err)
	}
	return nil
}
