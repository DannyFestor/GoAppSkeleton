// Package ui embeds the HTML templates and static assets, so the built binary is the whole deployment.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed templates static
var files embed.FS

var (
	Templates = mustSub(files, "templates")
	// Static keeps its "static" prefix so it serves directly under the /static/ route.
	Static fs.FS = files
)

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}

	return sub
}
