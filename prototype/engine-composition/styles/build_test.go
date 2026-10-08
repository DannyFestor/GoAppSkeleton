//go:build build

package styles_test

import (
	"os/exec"
	"testing"

	"example.com/enginecomp/config"
	"example.com/enginecomp/styles"
)

// `go mod tidy` runs here, at test time, only to obtain go.sum; how a Skeleton ships go.sum is still open on the map.
func TestEveryConfigurationBuilds(t *testing.T) {
	for _, cfg := range config.All("example.com/myapp") {
		t.Run(cfg.Name(), func(t *testing.T) {
			t.Parallel()
			tree, err := styles.All[styles.Control](cfg)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := tree.WriteTo(dir); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				cmd := exec.Command("go", args...)
				cmd.Dir = dir
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("go %v: %v\n%s", args, err, output)
				}
			}
		})
	}
}
