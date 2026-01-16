package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geoNotifications/internal/domain"
	retry "geoNotifications/internal/rerty"
	"geoNotifications/internal/webhookSender"

	"go.uber.org/zap/zapcore"

	"time"
)

func (r *Redis) EnqueueLocationCheck(taskData domain.LocationCheckTask, ctx context.Context, timeOut time.Duration) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeOut)
	defer cancel()
	data, err := json.Marshal(taskData)
	if err != nil {
		return fmt.Errorf("marshal location task: %w", err)
	}
	err = r.client.RPush(ctxWithTimeout, "location_check_queue", data).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) Run(ctx context.Context, webhookSender *webhookSender.WebhookSender) {
	for {
		select {
		case <-ctx.Done():
			r.logger.Log(zapcore.InfoLevel, "queue worker stopped")
			return
		default:
		}

		data, err := r.client.BRPop(ctx, 0, "location_check_queue").Result()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			r.logger.Log(zapcore.ErrorLevel, "queue read error: "+err.Error())
			continue
		}

		var locationTask domain.LocationCheckTask
		if err := json.Unmarshal([]byte(data[1]), &locationTask); err != nil {
			r.logger.Log(zapcore.ErrorLevel, "unmarshal task: "+err.Error())
			continue
		}

		err = retry.DoContext(ctx, retry.Strategy{
			Attempts: r.cfg.Retry.MaxAttempts,
			Delay:    r.cfg.Retry.Delay,
			Backoff:  r.cfg.Retry.Backoff,
		}, func() error {
			return webhookSender.SendWebhook(locationTask)
		})

		if err != nil {
			r.logger.Log(zapcore.ErrorLevel, "send webhook failed: "+err.Error())
		}
	}
}

func (r *Redis) StartQueue(ctx context.Context, webhookSender *webhookSender.WebhookSender) {
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.Run(ctx, webhookSender)
	}()
}

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
