package redis

import (
	"context"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"sync"
)

// CacheRetryQueue retries failed cache writes in a background worker.
type CacheRetryQueue struct {
	queue  chan *domain.Incident
	wg     sync.WaitGroup
	cancel context.CancelFunc
	cfg    *config.App
}

// NewCacheRetryQueue creates a buffered retry queue.
func NewCacheRetryQueue(cfg *config.App) *CacheRetryQueue {
	return &CacheRetryQueue{
		queue: make(chan *domain.Incident, cfg.DB.Redis.CacheRetrySize),
		cfg:   cfg,
	}
}

// Enqueue adds an incident to the retry queue. Non-blocking; drops if full.
func (q *CacheRetryQueue) Enqueue(incident *domain.Incident) {
	select {
	case q.queue <- incident:
	default:
	}
}

// Run processes the retry queue until ctx is cancelled.
func (q *CacheRetryQueue) Run(ctx context.Context, r *Redis) {
	for {
		select {
		case inc := <-q.queue:
			if err := r.Set(ctx, inc, q.cfg.DB.TimeOuts.Write); err != nil {
				q.Enqueue(inc)
			}
		case <-ctx.Done():
			return
		}
	}
}

// StartRetryWorker launches the retry worker in a goroutine.
func (q *CacheRetryQueue) StartRetryWorker(ctx context.Context, r *Redis) {
	ctx, cancel := context.WithCancel(ctx)
	q.cancel = cancel

	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		q.Run(ctx, r)
	}()
}

// StopRetryWorker signals the worker to stop and waits for completion.
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
