package httpx

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

// Chain wraps handler so the first middleware listed is the outermost, matching the order a request passes through them.
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		handler = middleware(handler)
	}

	return handler
}
