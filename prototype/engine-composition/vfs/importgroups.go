package vfs

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// regroupImports rewrites the single import block as standard library, third-party, then module-local groups,
// so a fragment can contribute an import without knowing which group it lands in. Import comments are not preserved.
func regroupImports(src []byte, module string) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ImportsOnly)
	if err != nil || len(file.Imports) == 0 {
		return src, err
	}
	decl := file.Decls[0].(*ast.GenDecl)
	start, end := fset.Position(decl.Pos()).Offset, fset.Position(decl.End()).Offset

	var out bytes.Buffer
	out.Write(src[:start])
	out.WriteString(importBlock(file.Imports, module))
	out.Write(src[end:])
	return out.Bytes(), nil
}

func importBlock(specs []*ast.ImportSpec, module string) string {
	groups := make([][]string, 3)
	for _, spec := range specs {
		group := importGroup(mustUnquote(spec.Path.Value), module)
		groups[group] = append(groups[group], importLine(spec))
	}
	var nonEmpty []string
	for _, group := range groups {
		if len(group) > 0 {
			nonEmpty = append(nonEmpty, "\t"+strings.Join(group, "\n\t"))
		}
	}
	return "import (\n" + strings.Join(nonEmpty, "\n\n") + "\n)"
}

const (
	standardLibrary = iota
	thirdParty
	moduleLocal
)

func importGroup(importPath, module string) int {
	switch {
	case importPath == module || strings.HasPrefix(importPath, module+"/"):
		return moduleLocal
	case strings.Contains(strings.Split(importPath, "/")[0], "."):
		return thirdParty
	default:
		return standardLibrary
	}
}

func importLine(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name + " " + spec.Path.Value
	}
	return spec.Path.Value
}
