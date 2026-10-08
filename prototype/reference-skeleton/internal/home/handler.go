package home

import (
	"net/http"

	"example.com/myapp/internal/httpx"
)

const page = "home.html"

type responder interface {
	Handle(fn httpx.HandlerFunc) http.Handler
	Render(w http.ResponseWriter, status int, page string, data any) error
}

type pageData struct {
	Title string
}

type Handler struct {
	responder responder
}

func New(responder responder) *Handler {
	return &Handler{responder: responder}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /{$}", h.responder.Handle(h.show))
}

func (h *Handler) show(w http.ResponseWriter, _ *http.Request) error {
	return h.responder.Render(w, http.StatusOK, page, pageData{Title: "Home"})
}
