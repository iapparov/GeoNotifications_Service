package redis

import (
	"context"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"sync"
)

type CacheRetryQueue struct {
	queue  chan *domain.Incident
	wg     sync.WaitGroup
	cancel context.CancelFunc
	config *config.App
}

func NewCacheRetryQueue(cfg *config.App) *CacheRetryQueue {
	return &CacheRetryQueue{queue: make(chan *domain.Incident, cfg.DB.Redis.CacheRetrySize), config: cfg}
}

func (q *CacheRetryQueue) Enqueue(incident *domain.Incident) {
	select {
	case q.queue <- incident:
	default:
	}
}

func (q *CacheRetryQueue) Run(ctx context.Context, redis *Redis) {
	for {
		select {
		case incident := <-q.queue:
			if err := redis.Set(incident, ctx, q.config.DB.TimeOuts.Write); err != nil {
				q.Enqueue(incident)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (q *CacheRetryQueue) StartRetryWorker(ctx context.Context, redis *Redis) {
	ctx, cancel := context.WithCancel(ctx)
	q.cancel = cancel

	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		q.Run(ctx, redis)
	}()
}

func (q *CacheRetryQueue) StopRetryWorker(ctx context.Context) error {
	if q.cancel != nil {
		q.cancel()
	}
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
