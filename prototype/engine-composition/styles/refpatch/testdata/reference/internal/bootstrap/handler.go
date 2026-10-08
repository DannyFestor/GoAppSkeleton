package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
)

type Dependencies struct {
	Logger *slog.Logger
}

// NewHandler is the composition root: every route and middleware is wired here, and nowhere else.
func NewHandler(deps Dependencies) http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /healthz", health(deps))
	router.HandleFunc("GET /{$}", home())

	return router
}

func health(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}
}

func home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	}
}
