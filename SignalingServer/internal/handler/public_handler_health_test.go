package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"

	"github.com/gin-gonic/gin"
)

func TestHealthNamesSQLiteComponent(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "health.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	redisRuntime, err := database.InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime(): %v", err)
	}
	t.Cleanup(func() { _ = redisRuntime.Close() })

	handler := NewPublicHandler(db, redisRuntime.Client, nil, nil, nil, "test")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", handler.Health)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Components map[string]string `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if got := payload.Components["sqlite"]; got != "ok" {
		t.Fatalf("sqlite component = %q, want ok; components=%v", got, payload.Components)
	}
	if _, exists := payload.Components["postgres"]; exists {
		t.Fatalf("SQLite health response still reports a postgres component: %v", payload.Components)
	}
}
