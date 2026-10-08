package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantCode int
	}{
		{
			name:     "invalid config fails before serving",
			env:      map[string]string{"LOG_LEVEL": "loud"},
			wantCode: exitFailure,
		},
		{
			name:     "cancelled context shuts down cleanly",
			env:      map[string]string{"ADDR": "127.0.0.1:0"},
			wantCode: exitOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			var stdout, stderr bytes.Buffer
			env := environment{
				stdout: &stdout,
				stderr: &stderr,
				getenv: func(key string) string { return tt.env[key] },
			}

			if code := run(ctx, env); code != tt.wantCode {
				t.Errorf("code = %d, want %d; stderr: %s; stdout: %s", code, tt.wantCode, &stderr, &stdout)
			}
		})
	}
}
