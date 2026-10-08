package httpx

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

const (
	layoutPattern = "layouts/*.html"
	pagePattern   = "pages/*.html"
	layoutName    = "base"
)

// Renderer holds every page parsed once at startup, so a broken template fails the boot, not a request.
type Renderer struct {
	pages map[string]*template.Template
}

func NewRenderer(templates fs.FS) (*Renderer, error) {
	pagePaths, err := fs.Glob(templates, pagePattern)
	if err != nil {
		return nil, err
	}

	pages := make(map[string]*template.Template, len(pagePaths))
	for _, pagePath := range pagePaths {
		page, err := template.ParseFS(templates, layoutPattern, pagePath)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", pagePath, err)
		}

		pages[path.Base(pagePath)] = page
	}

	return &Renderer{pages: pages}, nil
}

// Render executes into a buffer first so a failing template never sends a half-written page with a 200 status.
func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data any) error {
	tmpl, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("page %q not found", page)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, layoutName, data); err != nil {
		return fmt.Errorf("rendering %s: %w", page, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)

	return err
}
