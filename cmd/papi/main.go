package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	v1 "github.com/rabobank/papi/internal/api/v1"
	"github.com/rabobank/papi/internal/api/v1/handlers"
	"github.com/rabobank/papi/internal/api/v1/middleware"
	"github.com/rabobank/papi/internal/config"
	"github.com/rabobank/papi/internal/domain/group"
	"github.com/rabobank/papi/internal/health"
	"github.com/rabobank/papi/internal/store/migrations"
	"github.com/rabobank/papi/internal/store/postgres"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: papi <serve|migrate> [args]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := runServe(); err != nil {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	case "migrate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: papi migrate <up|down>")
			os.Exit(1)
		}
		if err := runMigrate(os.Args[2]); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runServe() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := config.SetupLogging(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pool, err := postgres.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	// OIDC Provider
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return fmt.Errorf("initializing OIDC provider: %w", err)
	}
	verifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})

	// Stores
	groupStore := postgres.NewGroupStore(pool)
	roleStore := postgres.NewRoleStore(pool)

	// Services
	healthService := health.NewService(pool)
	groupService := group.NewService(groupStore)

	// Handlers
	healthAdapter := handlers.NewHealthServiceAdapter(healthService)
	systemHandler := handlers.NewSystemHandler(healthAdapter, "0.1.0", cfg.OIDCIssuer)
	namespaceHandler := handlers.NewNamespaceHandler(nil, nil, roleStore)
	groupHandler := handlers.NewGroupHandler(groupService, roleStore)

	// Rate limiter (100 rps, burst 200)
	rateLimiter := middleware.NewRateLimiter(100, 200, logger)

	// Router
	router := v1.NewRouter(v1.RouterConfig{
		Verifier:         verifier,
		Logger:           logger,
		SystemHandler:    systemHandler,
		NamespaceHandler: namespaceHandler,
		GroupHandler:     groupHandler,
		RateLimiter:      rateLimiter,
	})

	// HTTP Server
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	logger.Info("starting PAPI server", "addr", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func runMigrate(direction string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.DBUrl)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	switch direction {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		fmt.Println("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		fmt.Println("Migrations rolled back successfully")
	default:
		return fmt.Errorf("unknown direction: %s (use 'up' or 'down')", direction)
	}
	return nil
}
