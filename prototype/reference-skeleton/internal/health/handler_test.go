package health_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/myapp/internal/health"
)

func TestHealthReportsOK(t *testing.T) {
	mux := http.NewServeMux()
	health.New().Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("got %d %q, want 200 \"ok\"", rec.Code, rec.Body.String())
	}
}
