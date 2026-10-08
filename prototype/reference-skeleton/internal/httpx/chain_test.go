package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"example.com/myapp/internal/httpx"
)

func TestChainRunsMiddlewaresInListedOrder(t *testing.T) {
	var visited []string
	record := func(name string) httpx.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				visited = append(visited, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { visited = append(visited, "handler") })

	httpx.Chain(handler, record("first"), record("second")).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if want := []string{"first", "second", "handler"}; !slices.Equal(visited, want) {
		t.Errorf("visited %v, want %v", visited, want)
	}
}
