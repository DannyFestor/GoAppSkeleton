package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = time.Minute
	shutdownTimeout   = 10 * time.Second
)

// Serve listens on addr until ctx is cancelled, then drains in-flight requests before returning.
func Serve(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	listenErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", addr)
		listenErr <- server.ListenAndServe()
	}()

	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
		return shutdown(server, logger)
	}
}

func shutdown(server *http.Server, logger *slog.Logger) error {
	logger.Info("shutting down")

	// ctx is already cancelled, so draining needs a fresh deadline of its own.
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
