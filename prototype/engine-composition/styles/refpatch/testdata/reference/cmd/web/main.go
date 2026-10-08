package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/myapp/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(ctx, logger); err != nil {
		logger.Error("exiting", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	deps := bootstrap.Dependencies{
		Logger: logger,
	}

	server := &http.Server{Addr: ":4000", Handler: bootstrap.NewHandler(deps)}
	go shutdownOnCancel(ctx, server)

	logger.Info("listening", "addr", server.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdownOnCancel(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	_ = server.Shutdown(context.Background())
}
