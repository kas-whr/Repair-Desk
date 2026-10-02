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

	"github.com/kas-whr/Repair-Desk/internal/config"
	"github.com/kas-whr/Repair-Desk/internal/httpapi"
	"github.com/kas-whr/Repair-Desk/internal/repository/postgres"
	"github.com/kas-whr/Repair-Desk/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	pool, err := postgres.NewPool(connectCtx, cfg.DB.DSN())
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()

	now := time.Now
	categoryRepo := postgres.NewCategoryRepository(pool)
	equipmentRepo := postgres.NewEquipmentRepository(pool)
	ticketRepo := postgres.NewTicketRepository(pool)
	commentRepo := postgres.NewCommentRepository(pool)

	router := httpapi.NewRouter(httpapi.Deps{
		Categories: service.NewCategoryService(categoryRepo, now),
		Equipment:  service.NewEquipmentService(equipmentRepo),
		Tickets:    service.NewTicketService(ticketRepo, categoryRepo, equipmentRepo, now),
		Comments:   service.NewCommentService(commentRepo, ticketRepo, now),
		DB:         pool,
		Logger:     logger,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server started", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	// Graceful shutdown (12-factor IX): finish in-flight requests, then close the pool.
	logger.Info("shutting down")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}
