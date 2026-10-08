// Package overlay is the style B.1 sub-experiment, scoped to handler.go: a base Template of {{block}} defaults,
// cloned, then each selected Choice's overlay parsed on top so its {{define}}s replace the blocks (last wins).
// It is not registered in styles.All because it renders one file only.
package overlay

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"text/template"

	"example.com/enginecomp/config"
	"example.com/enginecomp/vfs"
)

//go:embed templates
var templates embed.FS

const target = "internal/bootstrap/handler.go"

var base = template.Must(template.New("handler.go").Funcs(template.FuncMap{"args": args}).ParseFS(templates, "templates/handler.go.tmpl"))

func RenderHandler(cfg config.Configuration) (vfs.Tree, error) {
	tmpl, err := base.Clone()
	if err != nil {
		return nil, err
	}
	if err := parseOverlays(tmpl, cfg); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "handler.go.tmpl", cfg); err != nil {
		return nil, err
	}
	tree := vfs.Tree{target: buf.Bytes(), "go.mod": []byte("module " + cfg.Module + "\n")}
	return tree, tree.Finish()
}

func parseOverlays(tmpl *template.Template, cfg config.Configuration) error {
	for _, option := range config.Catalogue {
		src, err := fs.ReadFile(templates, "templates/overlays/"+option.Name+"-"+cfg.Choice(option.Name)+".tmpl")
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := tmpl.Parse(string(src)); err != nil {
			return err
		}
	}
	return nil
}

func args(pairs ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}
