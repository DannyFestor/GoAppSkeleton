package config_test

import (
	"log/slog"
	"testing"

	"example.com/myapp/internal/config"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: config.Config{Addr: ":4000", LogLevel: slog.LevelInfo},
		},
		{
			name: "overrides",
			env:  map[string]string{"ADDR": ":8080", "LOG_LEVEL": "debug"},
			want: config.Config{Addr: ":8080", LogLevel: slog.LevelDebug},
		},
		{
			name:    "invalid log level",
			env:     map[string]string{"LOG_LEVEL": "loud"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.Load(func(key string) string { return tt.env[key] })

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
