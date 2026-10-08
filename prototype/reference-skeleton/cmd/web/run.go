package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"example.com/myapp/internal/bootstrap"
	"example.com/myapp/internal/config"
	"example.com/myapp/internal/httpx"
)

const (
	exitOK      = 0
	exitFailure = 1
)

func run(ctx context.Context, env environment) int {
	cfg, err := config.Load(env.getenv)
	if err != nil {
		fmt.Fprintln(env.stderr, err)
		return exitFailure
	}

	logger := newLogger(env, cfg)

	handler, err := bootstrap.NewHandler(logger, os.DirFS(cfg.StaticDir))
	if err != nil {
		logger.Error("building handler", "error", err)
		return exitFailure
	}

	if err := httpx.Serve(ctx, cfg.Addr, handler, logger); err != nil {
		logger.Error("serving", "error", err)
		return exitFailure
	}

	return exitOK
}

func newLogger(env environment, cfg config.Config) *slog.Logger {
	return slog.New(slog.NewTextHandler(env.stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
}
