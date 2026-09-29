package main

import (
	"cellapp/apps/server/internal/hosting"
	"cellapp/apps/server/migrations"
	"context"
	"encoding/json"
	"flag"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	migrate := flag.Bool("migrate", false, "Apply database migrations and exit")
	cleanup := flag.Bool("cleanup", false, "Clean expired artifacts and exit")
	suspend := flag.String("suspend", "", "Suspend application by ID")
	metrics := flag.Bool("metrics", false, "Print aggregate usage counters")
	flag.Parse()
	c, e := hosting.LoadConfig(os.Getenv)
	if e != nil {
		return e
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, e := pgxpool.New(ctx, c.DatabaseURL)
	if e != nil {
		return e
	}
	defer db.Close()
	if *migrate {
		return migrations.Run(ctx, db)
	}
	storage, e := hosting.NewStorage(c)
	if e != nil {
		return e
	}
	server := &hosting.Server{Config: c, DB: db, Storage: storage, Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
	if *cleanup {
		return server.Cleanup(ctx)
	}
	if *suspend != "" {
		return server.Suspend(ctx, *suspend)
	}
	if *metrics {
		var apps, bytes int64
		e = db.QueryRow(ctx, `SELECT (SELECT count(*) FROM apps WHERE NOT deleted),(SELECT COALESCE(sum(bytes),0) FROM deployments WHERE status<>'cleaned')`).Scan(&apps, &bytes)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]int64{"apps": apps, "reservedStorageBytes": bytes})
	}
	if c.AuthMode == "github" {
		server.Identity = hosting.NewIdentity(c)
	}
	server.Web, e = hosting.LoadWebFS(c.WebRoot)
	if e != nil {
		return e
	}
	httpServer := &http.Server{Addr: c.Address, Handler: server.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if e := server.Cleanup(ctx); e != nil {
					server.Logger.Error("cleanup_failed")
				}
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	server.Logger.Info("listening", "address", c.Address)
	e = httpServer.ListenAndServe()
	if e == http.ErrServerClosed {
		return nil
	}
	return e
}
