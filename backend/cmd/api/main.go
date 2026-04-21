package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ezdeploy/backend/internal/auth"
	"ezdeploy/backend/internal/config"
	"ezdeploy/backend/internal/db"
	"ezdeploy/backend/internal/httpapi"
	"ezdeploy/backend/internal/logging"
	"ezdeploy/backend/internal/middleware"
	"ezdeploy/backend/internal/migrate"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(parent context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := migrate.Apply(ctx, pool, "migrations")
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		logger.Info("applied migrations", "count", len(applied))
	}

	authService, err := auth.New(pool, cfg.SessionSecret, cfg.AppEnv)
	if err != nil {
		return err
	}

	handler := httpapi.New(pool, authService, cfg.StorageRoot)
	corsHandler := middleware.CORS(cfg.CORSOrigins)(handler)
	
	server := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           corsHandler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("backend listening", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
