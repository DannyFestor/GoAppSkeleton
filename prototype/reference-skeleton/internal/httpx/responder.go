package httpx

import (
	"errors"
	"log/slog"
	"net/http"
)

const errorPage = "error.html"

// HandlerFunc is the handler shape every feature slice uses: return an error instead of writing one.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

type pageRenderer interface {
	Render(w http.ResponseWriter, status int, page string, data any) error
}

type errorPageData struct {
	Status  int
	Message string
}

// Responder is the single place handler errors become responses, so slices never format error pages themselves.
type Responder struct {
	renderer pageRenderer
	logger   *slog.Logger
}

func NewResponder(renderer pageRenderer, logger *slog.Logger) *Responder {
	return &Responder{renderer: renderer, logger: logger}
}

func (rs *Responder) Render(w http.ResponseWriter, status int, page string, data any) error {
	return rs.renderer.Render(w, status, page, data)
}

// Handle adapts an error-returning HandlerFunc to http.Handler.
func (rs *Responder) Handle(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			rs.Error(w, r, err)
		}
	})
}

func (rs *Responder) Error(w http.ResponseWriter, r *http.Request, err error) {
	if statusErr, ok := errors.AsType[StatusError](err); ok {
		rs.writeErrorPage(w, r, statusErr)
		return
	}

	rs.logger.Error("handling request", "method", r.Method, "path", r.URL.Path, "error", err)
	rs.writeErrorPage(w, r, StatusError{
		Status:  http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	})
}

func (rs *Responder) writeErrorPage(w http.ResponseWriter, r *http.Request, statusErr StatusError) {
	data := errorPageData{Status: statusErr.Status, Message: statusErr.Message}
	if err := rs.renderer.Render(w, statusErr.Status, errorPage, data); err != nil {
		// The error page itself failed, so fall back to plain text rather than recursing.
		rs.logger.Error("rendering error page", "path", r.URL.Path, "error", err)
		http.Error(w, statusErr.Message, statusErr.Status)
	}
}
