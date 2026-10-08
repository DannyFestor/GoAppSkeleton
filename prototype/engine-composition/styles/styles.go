// Package styles lists the competing engines so the render command and the tests can treat them alike.
package styles

import (
	"example.com/enginecomp/config"
	"example.com/enginecomp/styles/wholefile"
	"example.com/enginecomp/vfs"
)

type Renderer func(config.Configuration) (vfs.Tree, error)

// Control is the style every other style must match byte for byte.
const Control = "wholefile"

var All = map[string]Renderer{
	"wholefile": wholefile.Render,
}
