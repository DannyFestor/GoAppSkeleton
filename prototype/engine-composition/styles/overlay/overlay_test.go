package overlay_test

import (
	"bytes"
	"testing"

	"example.com/enginecomp/config"
	"example.com/enginecomp/styles/overlay"
	"example.com/enginecomp/styles/wholefile"
)

func TestHandlerMatchesControl(t *testing.T) {
	const path = "internal/bootstrap/handler.go"
	for _, cfg := range config.All("example.com/myapp") {
		t.Run(cfg.Name(), func(t *testing.T) {
			want, err := wholefile.Render(cfg)
			if err != nil {
				t.Fatal(err)
			}
			got, err := overlay.RenderHandler(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(want[path], got[path]) {
				t.Errorf("--- want\n%s\n--- got\n%s", want[path], got[path])
			}
		})
	}
}
