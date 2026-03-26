package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/retry"
	"geoNotifications/internal/webhook"
	"time"

	"go.uber.org/zap/zapcore"
)

// EnqueueLocationCheck pushes a location check task to the Redis queue.
func (r *Redis) EnqueueLocationCheck(ctx context.Context, task domain.LocationCheckTask, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal location task: %w", err)
	}

	if err := r.client.RPush(ctx, "location_check_queue", data).Err(); err != nil {
		return fmt.Errorf("enqueue location check: %w", err)
	}
	return nil
}

// Run processes the location check queue until ctx is cancelled.
func (r *Redis) Run(ctx context.Context, sender *webhook.Sender) {
	for {
		select {
		case <-ctx.Done():
			r.log.Log(zapcore.InfoLevel, "queue worker stopped")
			return
		default:
		}

		data, err := r.client.BRPop(ctx, 0, "location_check_queue").Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			r.log.Log(zapcore.ErrorLevel, "queue read error: "+err.Error())
			continue
		}

		var task domain.LocationCheckTask
		if err := json.Unmarshal([]byte(data[1]), &task); err != nil {
			r.log.Log(zapcore.ErrorLevel, "unmarshal task: "+err.Error())
			continue
		}

		err = retry.Do(ctx, retry.Strategy{
			Attempts: r.cfg.Retry.MaxAttempts,
			Delay:    r.cfg.Retry.Delay,
			Backoff:  r.cfg.Retry.Backoff,
		}, func() error {
			return sender.Send(task)
		})
		if err != nil {
			r.log.Log(zapcore.ErrorLevel, "send webhook failed: "+err.Error())
		}
	}
}

// StartQueue launches the queue worker in a goroutine.
func (r *Redis) StartQueue(ctx context.Context, sender *webhook.Sender) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.Run(ctx, sender)
	}()
}

// StopQueue signals the queue worker to stop and waits for completion.
func (r *Redis) StopQueue(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
