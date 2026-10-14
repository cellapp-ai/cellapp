package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"cellapp/apps/server/internal/hosting"
	"github.com/jackc/pgx/v5/pgxpool"
)

func storage(prefix string) (*hosting.S3Storage, error) {
	config := hosting.Config{
		Production:        true,
		S3Endpoint:        os.Getenv(prefix + "_ENDPOINT"),
		S3Region:          os.Getenv(prefix + "_REGION"),
		S3Bucket:          os.Getenv(prefix + "_BUCKET"),
		S3AccessKeyID:     os.Getenv(prefix + "_ACCESS_KEY_ID"),
		S3SecretAccessKey: os.Getenv(prefix + "_SECRET_ACCESS_KEY"),
	}
	if config.S3Endpoint == "" || config.S3Region == "" || config.S3Bucket == "" || config.S3AccessKeyID == "" || config.S3SecretAccessKey == "" {
		return nil, fmt.Errorf("incomplete %s storage configuration", prefix)
	}
	return hosting.NewStorage(config)
}

func run(ctx context.Context) error {
	verifyOnly := flag.Bool("verify-only", false, "verify destination without writing")
	flag.Parse()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	seconds, err := strconv.ParseInt(os.Getenv("ROLLBACK_RETENTION_SECONDS"), 10, 64)
	if err != nil || seconds < 60 {
		return fmt.Errorf("ROLLBACK_RETENTION_SECONDS must be at least 60")
	}
	source, err := storage("COPY_SOURCE")
	if err != nil {
		return err
	}
	destination, err := storage("COPY_DEST")
	if err != nil {
		return err
	}
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	for _, target := range []hosting.Storage{source, destination} {
		if err := target.Health(ctx); err != nil {
			return fmt.Errorf("storage health failed: %w", err)
		}
	}
	count, err := hosting.CopyPublishedReleases(ctx, db, source, destination, time.Duration(seconds)*time.Second, *verifyOnly)
	if err != nil {
		return err
	}
	fmt.Printf("verified %d active or rollback-retained releases\n", count)
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "release copy failed:", err)
		os.Exit(1)
	}
}
