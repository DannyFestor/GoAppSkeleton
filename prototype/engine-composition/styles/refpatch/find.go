package refpatch

import (
	"github.com/dave/dst"
)

func funcDecl(file *dst.File, name string) *dst.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*dst.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	panic("refpatch: no func " + name)
}

func structType(file *dst.File, name string) *dst.StructType {
	var found *dst.StructType
	dst.Inspect(file, func(node dst.Node) bool {
		if spec, ok := node.(*dst.TypeSpec); ok && spec.Name.Name == name {
			found = spec.Type.(*dst.StructType)
		}
		return found == nil
	})
	if found == nil {
		panic("refpatch: no struct " + name)
	}
	return found
}

func compositeLit(node dst.Node, typeName string) *dst.CompositeLit {
	var found *dst.CompositeLit
	dst.Inspect(node, func(n dst.Node) bool {
		if lit, ok := n.(*dst.CompositeLit); ok {
			if ident, ok := lit.Type.(*dst.Ident); ok && ident.Name == typeName {
				found = lit
			}
		}
		return found == nil
	})
	return found
}

// returnedFuncLit is the handler closure that every handler constructor in the Skeleton returns.
func returnedFuncLit(fn *dst.FuncDecl) *dst.FuncLit {
	ret := fn.Body.List[len(fn.Body.List)-1].(*dst.ReturnStmt)
	return ret.Results[0].(*dst.FuncLit)
}

func methodCalls(node dst.Node, receiver, method string) []*dst.CallExpr {
	var calls []*dst.CallExpr
	dst.Inspect(node, func(n dst.Node) bool {
		if call, ok := n.(*dst.CallExpr); ok {
			if sel, ok := call.Fun.(*dst.SelectorExpr); ok && isIdent(sel.X, receiver) && sel.Sel.Name == method {
				calls = append(calls, call)
			}
		}
		return true
	})
	return calls
}

func isIdent(expr dst.Expr, name string) bool {
	ident, ok := expr.(*dst.Ident)
	return ok && ident.Name == name
}

func indexOfStmt(list []dst.Stmt, match func(dst.Stmt) bool) int {
	for i, stmt := range list {
		if match(stmt) {
			return i
		}
	}
	panic("refpatch: statement not found")
}

// insertAfter hands the anchor's trailing spacing to the inserted statement, so a blank line that closed a group still closes it.
func insertAfter(list []dst.Stmt, anchor int, stmt dst.Stmt) []dst.Stmt {
	stmt.Decorations().After = list[anchor].Decorations().After
	list[anchor].Decorations().After = dst.NewLine
	return insertStmts(list, anchor+1, stmt)
}

func insertStmts(list []dst.Stmt, at int, stmts ...dst.Stmt) []dst.Stmt {
	return append(list[:at], append(stmts, list[at:]...)...)
}

func isRoute(quotedPattern string) func(dst.Stmt) bool {
	return func(stmt dst.Stmt) bool {
		expr, ok := stmt.(*dst.ExprStmt)
		if !ok {
			return false
		}
		call, ok := expr.X.(*dst.CallExpr)
		if !ok || len(call.Args) == 0 {
			return false
		}
		pattern, ok := call.Args[0].(*dst.BasicLit)
		return ok && pattern.Value == quotedPattern
	}
}
