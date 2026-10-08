package home

import (
	"net/http"

	"example.com/myapp/internal/httpx"
)

const page = "home.html"

type pageData struct {
	Title string
}

type Handler struct {
	responder *httpx.Responder
}

func New(responder *httpx.Responder) *Handler {
	return &Handler{responder: responder}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /{$}", h.responder.Handle(h.show))
}

func (h *Handler) show(w http.ResponseWriter, _ *http.Request) error {
	return h.responder.Render(w, http.StatusOK, page, pageData{Title: "Home"})
}
