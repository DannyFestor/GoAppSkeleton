package httpx_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"example.com/myapp/internal/httpx"
)

var testTemplates = fstest.MapFS{
	"layouts/base.html": {Data: []byte(`{{define "base"}}{{template "main" .}}{{end}}`)},
	"pages/error.html":  {Data: []byte(`{{define "main"}}{{.Status}} {{.Message}}{{end}}`)},
}

func TestResponderError(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "status error is shown to the client",
			handlerErr: httpx.StatusError{Status: http.StatusBadRequest, Message: "bad input"},
			wantStatus: http.StatusBadRequest,
			wantBody:   "400 bad input",
		},
		{
			name:       "unexpected error is hidden behind a 500",
			handlerErr: errors.New("database password is hunter2"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responder := newTestResponder(t)
			rec := httptest.NewRecorder()
			responder.Error(rec, httptest.NewRequest(http.MethodGet, "/", nil), tt.handlerErr)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body := rec.Body.String(); !strings.Contains(body, tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tt.wantBody)
			}
		})
	}
}

func newTestResponder(t *testing.T) *httpx.Responder {
	t.Helper()

	renderer, err := httpx.NewRenderer(testTemplates)
	if err != nil {
		t.Fatal(err)
	}

	return httpx.NewResponder(renderer, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
