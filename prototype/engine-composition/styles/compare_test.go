package styles_test

import (
	"bytes"
	"testing"

	"example.com/enginecomp/config"
	"example.com/enginecomp/styles"
	"example.com/enginecomp/vfs"
)

func TestEveryStyleMatchesControl(t *testing.T) {
	for name, render := range styles.All {
		if name == styles.Control {
			continue
		}
		t.Run(name, func(t *testing.T) {
			for _, cfg := range config.All("example.com/myapp") {
				t.Run(cfg.Name(), func(t *testing.T) {
					assertSameTree(t, mustRender(t, styles.All[styles.Control], cfg), mustRender(t, render, cfg))
				})
			}
		})
	}
}

func mustRender(t *testing.T, render styles.Renderer, cfg config.Configuration) vfs.Tree {
	t.Helper()
	tree, err := render(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func assertSameTree(t *testing.T, want, got vfs.Tree) {
	t.Helper()
	for _, path := range want.Paths() {
		if !bytes.Equal(want[path], got[path]) {
			t.Errorf("%s differs\n--- want\n%s\n--- got\n%s", path, want[path], got[path])
		}
	}
	for _, path := range got.Paths() {
		if _, ok := want[path]; !ok {
			t.Errorf("%s is extra", path)
		}
	}
}
