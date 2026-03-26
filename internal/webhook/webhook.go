package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Sender delivers location check tasks to an external webhook endpoint.
type Sender struct {
	url    string
	client *http.Client
	log    *logger.Service
}

// NewSender creates a Sender for the given webhook URL.
func NewSender(url string, log *logger.Service) *Sender {
	return &Sender{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		log:    log,
	}
}

// Send posts taskData as JSON to the configured webhook URL.
func (s *Sender) Send(taskData domain.LocationCheckTask) error {
	data, err := json.Marshal(taskData)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.log.Log(zapcore.ErrorLevel, "failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, resp.Status)
	}

	s.log.Log(zapcore.InfoLevel, "webhook sent",
		zap.String("uid", taskData.Location.ID.String()),
	)
	return nil
}
