package refpatch

import (
	"strconv"
	"strings"

	"github.com/dave/dst"
)

func useChiRouter(files goFiles) error {
	newHandler := funcDecl(files[handlerPath], "NewHandler")
	routerDecl := newHandler.Body.List[0].(*dst.AssignStmt)
	routerDecl.Rhs[0] = snippetExpr(bootstrapPkg, "package bootstrap\n\nimport \"github.com/go-chi/chi/v5\"\n\nfunc _() {\n\tchi.NewRouter()\n}")
	for _, call := range methodCalls(newHandler, "router", "HandleFunc") {
		toChiRoute(call)
	}
	return nil
}

// "GET /{$}" is a ServeMux pattern: method and exact-match marker become chi's method name and plain path.
func toChiRoute(call *dst.CallExpr) {
	pattern := call.Args[0].(*dst.BasicLit)
	unquoted, _ := strconv.Unquote(pattern.Value)
	method, path, _ := strings.Cut(unquoted, " ")
	call.Fun.(*dst.SelectorExpr).Sel.Name = strings.ToUpper(method[:1]) + strings.ToLower(method[1:])
	pattern.Value = strconv.Quote(strings.TrimSuffix(path, "{$}"))
}
