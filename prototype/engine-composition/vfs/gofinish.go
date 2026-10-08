package vfs

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// FinishGo removes unused and duplicate imports, groups and sorts them, then gofmts, all in-process:
// imports.Process would shell out to `go env`.
func FinishGo(src []byte, module string) ([]byte, error) {
	src, err := regroupImports(src, module)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	pruneImports(fset, file)
	ast.SortImports(fset, file)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return format.Source(buf.Bytes())
}

func pruneImports(fset *token.FileSet, file *ast.File) {
	for _, spec := range unwantedImports(file) {
		astutil.DeleteNamedImport(fset, file, specName(spec), mustUnquote(spec.Path.Value))
	}
}

// Collected before deleting because DeleteNamedImport rewrites file.Imports.
func unwantedImports(file *ast.File) []*ast.ImportSpec {
	used := usedQualifiers(file)
	seen := map[string]bool{}
	var unwanted []*ast.ImportSpec
	for _, spec := range file.Imports {
		importPath := mustUnquote(spec.Path.Value)
		name := localName(spec)
		if name == "_" || name == "." {
			continue
		}
		if seen[importPath] || !used[name] {
			unwanted = append(unwanted, spec)
			continue
		}
		seen[importPath] = true
	}
	return unwanted
}

func usedQualifiers(file *ast.File) map[string]bool {
	used := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok {
				used[ident.Name] = true
			}
		}
		return true
	})
	return used
}

var majorVersionSuffix = regexp.MustCompile(`/v[0-9]+$`)

// Without loading packages the name is a guess from the path; astutil guesses "v5" for chi/v5, so strip the major version first.
// A Template importing a package whose name differs from its last path element must name the import explicitly.
func localName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	importPath := majorVersionSuffix.ReplaceAllString(mustUnquote(spec.Path.Value), "")
	return importPath[strings.LastIndex(importPath, "/")+1:]
}

func specName(spec *ast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	return ""
}

func mustUnquote(s string) string {
	unquoted, _ := strconv.Unquote(s)
	return unquoted
}
