// Package vfs is the in-memory Skeleton every style renders into; front ends would commit it to disk or a zip.
package vfs

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Tree map[string][]byte

func (t Tree) Paths() []string {
	return slices.Sorted(maps.Keys(t))
}

// Finish is the one post-render pass shared by all styles, so differences between styles come from the Templates alone.
func (t Tree) Finish() error {
	for _, path := range t.Paths() {
		if !strings.HasSuffix(path, ".go") {
			t[path] = withSingleTrailingNewline(t[path])
			continue
		}
		finished, err := FinishGo(t[path])
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", path, err, t[path])
		}
		t[path] = finished
	}
	return nil
}

func withSingleTrailingNewline(content []byte) []byte {
	return append(bytes.TrimRight(content, "\n"), '\n')
}

func (t Tree) WriteTo(dir string) error {
	for _, path := range t.Paths() {
		target := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, t[path], 0o644); err != nil {
			return err
		}
	}
	return nil
}
