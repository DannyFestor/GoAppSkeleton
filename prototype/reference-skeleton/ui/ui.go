// Package ui embeds the HTML templates, so a broken or missing view is caught at build and boot rather than per deployment.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed templates
var files embed.FS

var Templates = mustSub(files, "templates")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}

	return sub
}
