package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	code := run(ctx, processEnvironment())

	stop()
	os.Exit(code)
}

func processEnvironment() environment {
	return environment{
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
	}
}
