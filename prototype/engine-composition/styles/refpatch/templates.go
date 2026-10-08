package refpatch

import (
	"bytes"
	"io/fs"
	"strings"
	"text/template"

	"example.com/enginecomp/config"
	"example.com/enginecomp/vfs"
)

// Non-Go files have no AST to patch, so they fall back to style A's whole-file conditional Templates.
func renderTemplates(tree vfs.Tree, root fs.FS, cfg config.Configuration) error {
	return fs.WalkDir(root, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rendered, err := renderTemplate(root, path, cfg)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(rendered)) > 0 {
			tree[strings.TrimSuffix(path, ".tmpl")] = rendered
		}
		return nil
	})
}

func renderTemplate(root fs.FS, path string, cfg config.Configuration) ([]byte, error) {
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
