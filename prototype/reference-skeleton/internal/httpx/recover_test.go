package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/myapp/internal/httpx"
)

type fakeErrorResponder struct {
	reported error
}

func (f *fakeErrorResponder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	f.reported = err
	w.WriteHeader(http.StatusInternalServerError)
}

func TestRecoverReportsPanic(t *testing.T) {
	responder := &fakeErrorResponder{}
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	httpx.Recover(responder, panicking).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if responder.reported == nil {
		t.Fatal("panic was not reported")
	}
	if rec.Header().Get("Connection") != "close" {
		t.Error("connection was not marked for closing")
	}
}
