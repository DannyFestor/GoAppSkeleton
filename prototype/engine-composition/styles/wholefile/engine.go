// Package wholefile is style A: one Template per Skeleton file, every Option axis as an inline {{if}}.
package wholefile

import (
	"bytes"
	"embed"
	"io/fs"
	"strings"
	"text/template"

	"example.com/enginecomp/config"
	"example.com/enginecomp/vfs"
)

//go:embed all:templates
var templates embed.FS

func Render(cfg config.Configuration) (vfs.Tree, error) {
	root, _ := fs.Sub(templates, "templates")
	tree := vfs.Tree{}
	err := fs.WalkDir(root, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rendered, err := renderFile(root, path, cfg)
		if err != nil {
			return err
		}
		// An empty render is how a Template says "this file is not part of the Configuration".
		if len(bytes.TrimSpace(rendered)) > 0 {
			tree[strings.TrimSuffix(path, ".tmpl")] = rendered
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tree, tree.Finish()
}

func renderFile(root fs.FS, path string, cfg config.Configuration) ([]byte, error) {
	src, err := fs.ReadFile(root, path)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(path).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, cfg)
	return buf.Bytes(), err
}
