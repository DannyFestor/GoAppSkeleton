// Package refpatch is style D: a compiling reference Skeleton, edited per Choice by dst patches written in Go.
package refpatch

import (
	"embed"
	"io/fs"

	"example.com/enginecomp/config"
	"example.com/enginecomp/vfs"
)

// The reference lives under testdata because embed refuses a directory holding its own go.mod,
// and without one its packages would compile as part of the Generator's module.
//
//go:embed testdata/reference testdata/overlays
var goSources embed.FS

//go:embed all:templates
var templates embed.FS

func Render(cfg config.Configuration) (vfs.Tree, error) {
	files, err := loadGoFiles(mustSub(goSources, "testdata/reference"))
	if err != nil {
		return nil, err
	}
	if err := applyPatches(files, cfg); err != nil {
		return nil, err
	}
	tree, err := files.restore(cfg.Module)
	if err != nil {
		return nil, err
	}
	if err := renderTemplates(tree, mustSub(templates, "templates"), cfg); err != nil {
		return nil, err
	}
	return tree, tree.Finish()
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	root, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return root
}
