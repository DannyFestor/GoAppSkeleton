// Package slots is style B: base Templates declare named slots; each selected Choice ships a fragment set that fills them.
//
// Layout under templates/:
//
//	base/<path>.tmpl                 one per Skeleton file, always rendered
//	<option>/_any/                   applies when the Option's Choice is anything but none
//	<option>/<choice>/               applies when that Choice is selected
//	  fragments.tmpl                 {{define "<slot>"}} contributions and {{define "<provider>"}} single definitions
//	  files/<path>.tmpl              whole Skeleton files only this Choice adds
package slots

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"example.com/enginecomp/config"
	"example.com/enginecomp/vfs"
)

//go:embed all:templates
var templates embed.FS

const anyChoice = "_any"

type fragmentSet struct {
	dir  string
	tmpl *template.Template
}

type renderer struct {
	cfg  config.Configuration
	root fs.FS
	sets []fragmentSet
}

func Render(cfg config.Configuration) (vfs.Tree, error) {
	root, _ := fs.Sub(templates, "templates")
	r := &renderer{cfg: cfg, root: root}
	if err := r.loadFragmentSets(); err != nil {
		return nil, err
	}
	tree, err := r.renderAllFiles()
	if err != nil {
		return nil, err
	}
	return tree, tree.Finish()
}

// selectedDirs is in catalogue order, which is therefore the order contributions appear within every slot.
func (r *renderer) selectedDirs() []string {
	var dirs []string
	for _, option := range config.Catalogue {
		if r.cfg.Uses(option.Name) {
			dirs = append(dirs, path.Join(option.Name, anyChoice))
		}
		dirs = append(dirs, path.Join(option.Name, r.cfg.Choice(option.Name)))
	}
	return dirs
}

func (r *renderer) loadFragmentSets() error {
	for _, dir := range r.selectedDirs() {
		src, err := fs.ReadFile(r.root, path.Join(dir, "fragments.tmpl"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		tmpl, err := r.newTemplate(dir).Parse(string(src))
		if err != nil {
			return err
		}
		r.sets = append(r.sets, fragmentSet{dir: dir, tmpl: tmpl})
	}
	return nil
}

func (r *renderer) renderAllFiles() (vfs.Tree, error) {
	tree := vfs.Tree{}
	if err := r.renderFiles("base", tree); err != nil {
		return nil, err
	}
	for _, dir := range r.selectedDirs() {
		if err := r.renderFiles(path.Join(dir, "files"), tree); err != nil {
			return nil, err
		}
	}
	return tree, nil
}

func (r *renderer) renderFiles(dir string, tree vfs.Tree) error {
	if _, err := fs.Stat(r.root, dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return fs.WalkDir(r.root, dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		target := strings.TrimSuffix(strings.TrimPrefix(file, dir+"/"), ".tmpl")
		if _, taken := tree[target]; taken {
			return fmt.Errorf("%s: %s is already rendered by another fragment set", file, target)
		}
		rendered, err := r.renderFile(file)
		if err != nil {
			return err
		}
		// An empty render is how a base Template says "no Choice contributed, so this file is not needed".
		if len(bytes.TrimSpace(rendered)) > 0 {
			tree[target] = rendered
		}
		return nil
	})
}

func (r *renderer) renderFile(file string) ([]byte, error) {
	src, err := fs.ReadFile(r.root, file)
	if err != nil {
		return nil, err
	}
	tmpl, err := r.newTemplate(file).Parse(string(src))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, r.cfg)
	return buf.Bytes(), err
}

func (r *renderer) newTemplate(name string) *template.Template {
	return template.New(name).Option("missingkey=error").Funcs(r.funcs())
}
