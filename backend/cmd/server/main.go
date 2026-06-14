// Command server is the FitCoach backend entrypoint. It wires dependencies and
// starts the HTTP server; all business logic lives in internal packages.
//
// Usage:
//
//	server            start the HTTP server (runs pending migrations first)
//	server migrate    apply pending migrations and exit
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/config"
	"pro.d11l.fitcoach/backend/internal/platform/db"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
	"pro.d11l.fitcoach/backend/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}

func run(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(os.Stdout, cfg.LogLevel)

	ctx := context.Background()
	database, err := db.Open(ctx, cfg.MySQLDSN.Reveal())
	if err != nil {
		return err
	}
	defer database.Close()

	// `migrate` subcommand: apply migrations and exit.
	if len(args) > 0 && args[0] == "migrate" {
		if err := db.Migrate(ctx, database, migrations.FS); err != nil {
			return err
		}
		logger.Info("migrations applied")
		return nil
	}

	// On boot, ensure the schema is current before serving traffic.
	if err := db.Migrate(ctx, database, migrations.FS); err != nil {
		return err
	}

	authSvc := auth.NewService(auth.NewStore(database), auth.Config{
		JWTKey:     []byte(cfg.JWTSigningKey.Reveal()),
		AccessTTL:  cfg.AccessTokenTTL,
		RefreshTTL: cfg.RefreshTokenTTL,
	}, auth.NewLogMailer(logger), nil)
	authHandler := auth.NewHandler(authSvc, logger)

	router := httpx.NewRouter()
	router.Use(logging.Middleware(logger))
	router.HandleFunc("GET /healthz", httpx.Health())
	authHandler.Register(router)

	return serve(cfg, logger, router)
}

func serve(cfg config.Config, logger *logging.Logger, router http.Handler) error {

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start serving in the background so we can wait for a shutdown signal.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
