package appdata

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEnsureSeparate(t *testing.T) {
	controlURL, appURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_APP_DATA_DATABASE_URL")
	if controlURL == "" || appURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_APP_DATA_DATABASE_URL required")
	}
	ctx := context.Background()
	control, err := pgxpool.New(ctx, controlURL)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	application, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	if err := EnsureSeparate(ctx, control, application); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSeparate(ctx, control, control); err == nil {
		t.Fatal("same database accepted")
	}
}
