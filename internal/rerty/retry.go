package retry

import (
	"context"
	"time"
)

type Strategy struct {
	Attempts int           // Количество попыток.
	Delay    time.Duration // Начальная задержка между попытками.
	Backoff  float64       // Множитель для увеличения задержки.
}

func DoContext(ctx context.Context, strategy Strategy, fn func() error) error {
	delay := strategy.Delay
	var err error
	for i := 0; i < strategy.Attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay = time.Duration(float64(delay) * strategy.Backoff)
	}
	return err
}
