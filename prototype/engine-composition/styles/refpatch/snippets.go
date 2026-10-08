package refpatch

import (
	"github.com/dave/dst"
)

// Snippets are whole Go files so their imports resolve; the patch then lifts out the part it needs.
func snippetFile(pkgPath, src string) *dst.File {
	file, err := decorate(pkgPath, src)
	if err != nil {
		panic("refpatch snippet does not parse: " + err.Error())
	}
	return file
}

func snippetStmts(pkgPath, src string) []dst.Stmt {
	return funcDecl(snippetFile(pkgPath, src), "_").Body.List
}

func snippetFuncs(pkgPath, src string) []dst.Decl {
	var funcs []dst.Decl
	for _, decl := range snippetFile(pkgPath, src).Decls {
		if fn, ok := decl.(*dst.FuncDecl); ok {
			funcs = append(funcs, fn)
		}
	}
	return funcs
}

func snippetFields(pkgPath, src string) []*dst.Field {
	return structType(snippetFile(pkgPath, src), "_").Fields.List
}

func snippetExpr(pkgPath, src string) dst.Expr {
	stmts := snippetStmts(pkgPath, src)
	return stmts[0].(*dst.ExprStmt).X
}
