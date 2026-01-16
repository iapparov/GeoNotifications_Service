package webhookSender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"geoNotifications/internal/config"
	"geoNotifications/internal/domain"
	"geoNotifications/internal/logger"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type WebhookSender struct {
	url    string
	client *http.Client
	logger *logger.Service
}

func NewWebhookSender(cfg *config.App, logger *logger.Service) *WebhookSender {
	return &WebhookSender{
		url:    cfg.WebHook,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

func (w *WebhookSender) SendWebhook(taskData domain.LocationCheckTask) error {
	data, err := json.Marshal(taskData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			w.logger.Log(zapcore.ErrorLevel, "failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d with error: %s", resp.StatusCode, resp.Status)
	}
	w.logger.Log(zapcore.InfoLevel, "webhook sent successfully", zap.String("uid", taskData.Location.ID.String()))
	return nil
}
