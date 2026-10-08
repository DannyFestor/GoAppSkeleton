package home_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/myapp/internal/home"
)

type fakeResponder struct {
	renderedPage   string
	renderedStatus int
}

func (f *fakeResponder) Error(http.ResponseWriter, *http.Request, error) {}

func (f *fakeResponder) Render(_ http.ResponseWriter, status int, page string, _ any) error {
	f.renderedStatus, f.renderedPage = status, page
	return nil
}

func TestHomeRendersHomePage(t *testing.T) {
	responder := &fakeResponder{}
	mux := http.NewServeMux()
	home.New(responder).Register(mux)

	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if responder.renderedPage != "home.html" || responder.renderedStatus != http.StatusOK {
		t.Errorf("rendered %q with %d, want home.html with 200", responder.renderedPage, responder.renderedStatus)
	}
}
