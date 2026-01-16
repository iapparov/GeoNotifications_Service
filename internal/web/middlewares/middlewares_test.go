package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"geoNotifications/internal/config"
	"geoNotifications/internal/logger"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware("secret"))
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := perform(r, http.MethodGet, "/ping", nil, nil)
	if w1.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w1.Code)
	}

	w2 := perform(r, http.MethodGet, "/ping", nil, map[string]string{"X-API-Key": "bad"})
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 bad key, got %d", w2.Code)
	}

	w3 := perform(r, http.MethodGet, "/ping", nil, map[string]string{"X-API-Key": "secret"})
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}
}

func TestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.App{Logger: config.Logger{BuffSize: 10, Mode: "dev", Level: "debug"}}
	l := logger.NewService(cfg)
	r := gin.New()
	r.Use(LoggerMiddleware(l))
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := perform(r, http.MethodGet, "/ok", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func perform(r http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
