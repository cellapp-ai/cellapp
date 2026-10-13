package hosting

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHealthProbeUsesContainerHost(t *testing.T) {
	db, err := pgxpool.New(context.Background(), "postgres://cellapp:unused@127.0.0.1:1/cellapp?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{
		Config: Config{ControlOrigin: "https://console.cellapp.ai", AppsDomain: "ce11.app"},
		DB:     db,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	handler := s.Handler()
	probe := httptest.NewRecorder()
	handler.ServeHTTP(probe, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/health", nil))
	if probe.Code != http.StatusServiceUnavailable {
		t.Fatalf("health probe status = %d, want database failure 503", probe.Code)
	}
	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/", nil))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown host status = %d, want 404", unknown.Code)
	}
}
