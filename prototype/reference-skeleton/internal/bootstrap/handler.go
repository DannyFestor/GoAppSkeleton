package bootstrap

import (
	"io/fs"
	"log/slog"
	"net/http"

	"example.com/myapp/internal/health"
	"example.com/myapp/internal/home"
	"example.com/myapp/internal/httpx"
	"example.com/myapp/ui"
)

// NewHandler is the composition root: every feature slice is constructed and mounted here, and nowhere else.
func NewHandler(logger *slog.Logger, static fs.FS) (http.Handler, error) {
	renderer, err := httpx.NewRenderer(ui.Templates)
	if err != nil {
		return nil, err
	}

	responder := httpx.NewResponder(renderer, logger)
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	health.New().Register(mux)
	home.New(responder).Register(mux)

	return httpx.Chain(mux, httpx.Recover(responder)), nil
}
