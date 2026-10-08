package bootstrap

import (
	"log/slog"
	"net/http"

	"example.com/myapp/internal/health"
	"example.com/myapp/internal/home"
	"example.com/myapp/internal/httpx"
	"example.com/myapp/ui"
)

// NewHandler is the composition root: every feature slice is constructed and mounted here, and nowhere else.
func NewHandler(logger *slog.Logger) (http.Handler, error) {
	renderer, err := httpx.NewRenderer(ui.Templates)
	if err != nil {
		return nil, err
	}

	responder := httpx.NewResponder(renderer, logger)
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(ui.Static))
	health.New().Register(mux)
	home.New(responder).Register(mux)

	return httpx.Chain(mux, httpx.Recover(responder)), nil
}
