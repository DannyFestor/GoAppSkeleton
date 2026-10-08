//go:build feature

package feature_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/myapp/internal/bootstrap"
)

func TestRoutes(t *testing.T) {
	handler, err := bootstrap.NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodGet, "/static/css/main.css", http.StatusOK},
		{http.MethodGet, "/missing", http.StatusNotFound},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
