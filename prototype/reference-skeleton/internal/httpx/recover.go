package httpx

import (
	"fmt"
	"net/http"
)

// Recover turns a panicking handler into a logged 500 instead of a dropped connection.
func Recover(responder *Responder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			// net/http uses ErrAbortHandler to abort a response on purpose; it must keep propagating.
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}

			w.Header().Set("Connection", "close")
			responder.Error(w, r, fmt.Errorf("panic: %v", recovered))
		}()

		next.ServeHTTP(w, r)
	})
}
