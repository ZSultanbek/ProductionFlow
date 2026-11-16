// Command api is the ProductionFlow HTTP server.
//
// This file is wiring only: read config, open the pool, build the router, serve,
// shut down cleanly. No routes, no rules, no SQL — those live in internal/.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ZSultanbek/ProductionFlow/backend/internal/auth"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/config"
	apihttp "github.com/ZSultanbek/ProductionFlow/backend/internal/http"
	"github.com/ZSultanbek/ProductionFlow/backend/internal/store"
)

// Stamped by the linker; see the ldflags in the Makefile.
var (
	version = "dev"
	commit  = "none"
	builtAt = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg.Env)
	slog.SetDefault(logger)

	// Signal-aware root context: Ctrl-C or SIGTERM cancels it, which starts shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// A dead database is not a reason to refuse to boot: the client team needs to
	// be able to run the API on its own, and /api/health reports the real state.
	if err := db.Ping(ctx); err != nil {
		logger.Warn("database unreachable at startup, serving anyway", "error", err)
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: apihttp.NewRouter(apihttp.Deps{
			DB:           db,
			Users:        db,
			Projects:     db,
			Days:         db,
			Tokens:       auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL),
			Build:        apihttp.BuildInfo{Version: version, Commit: commit, BuiltAt: builtAt},
			Logger:       logger,
			PublicWebURL: cfg.PublicWebURL,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// ListenAndServe blocks, so it runs in a goroutine and reports back on a channel.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr, "version", version, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		logger.Info("shutting down", "timeout", cfg.ShutdownTimeout)
	}

	// Fresh context: the root one is already cancelled by the signal.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("stopped")
	return nil
}

func newLogger(env string) *slog.Logger {
	if env == "dev" {
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
