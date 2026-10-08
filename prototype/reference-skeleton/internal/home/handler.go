package home

import "net/http"

const page = "home.html"

type responder interface {
	Render(w http.ResponseWriter, status int, page string, data any) error
	Error(w http.ResponseWriter, r *http.Request, err error)
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
	mux.HandleFunc("GET /{$}", h.show)
}

func (h *Handler) show(w http.ResponseWriter, r *http.Request) {
	if err := h.responder.Render(w, http.StatusOK, page, pageData{Title: "Home"}); err != nil {
		h.responder.Error(w, r, err)
	}
}
