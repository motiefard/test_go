package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"card2sheba/internal/config"
	"card2sheba/internal/httpapi"
	"card2sheba/internal/inquiry"
	"card2sheba/internal/repository"
	"card2sheba/internal/zarinhub"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(".env")
	if err != nil {
		log.Error("config_load_failed", "error", "invalid configuration")
		os.Exit(1)
	}

	repo, err := repository.Open(cfg.SQLitePath, cfg.MigrationsPath)
	if err != nil {
		log.Error("database_open_failed", "error", "persistence_error")
		os.Exit(1)
	}
	defer repo.Close()

	client := zarinhub.NewClient(cfg.ZarinHubBaseURL, cfg.BearerToken(), cfg.ZarinHubTimeout)
	svc := inquiry.NewService(client, repo, log)
	h := httpapi.NewHandler(svc, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(h, cfg.CORSOrigin),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.ZarinHubTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("server_started", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server_failed", "error", "listen_error")
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("server_stopped")
}
