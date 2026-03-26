package retry

import (
	"context"
	"time"
)

// Strategy configures retry behaviour.
type Strategy struct {
	Attempts int           // Maximum number of attempts.
	Delay    time.Duration // Initial delay between attempts.
	Backoff  float64       // Multiplier applied to delay after each attempt.
}

// Do executes fn up to strategy.Attempts times, respecting ctx cancellation.
func Do(ctx context.Context, s Strategy, fn func() error) error {
	delay := s.Delay
	var err error
	for i := range s.Attempts {
		if err = fn(); err == nil {
			return nil
		}
		if i == s.Attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * s.Backoff)
	}
	return err
}
