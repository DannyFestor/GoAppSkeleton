package refpatch

import (
	"strings"

	"github.com/dave/dst"
)

func addCookieSessions(files goFiles) error {
	addDependency(files, "SessionKey []byte", `"os"`, `SessionKey: []byte(os.Getenv("SESSION_KEY"))`)
	return addSessions(files, "sessions.NewCookieStore(deps.SessionKey)", "cookie")
}

func addSQLSessions(files goFiles) error {
	return addSessions(files, "sessions.NewSQLStore(deps.DB)", "database")
}

func addSessions(files goFiles, store, choice string) error {
	handler := files[handlerPath]
	newHandler := funcDecl(handler, "NewHandler")
	createManager(newHandler, store)
	passManagerToHome(newHandler)
	registerLogout(newHandler)
	wrapInSessionMiddleware(newHandler)
	replaceHomeWithSessionHandlers(handler)
	if err := addOverlay(files, "sessions/common"); err != nil {
		return err
	}
	return addOverlay(files, "sessions/"+choice)
}

func createManager(newHandler *dst.FuncDecl, store string) {
	manager := snippetStmts(bootstrapPkg, `package bootstrap

import "example.com/myapp/internal/sessions"

func _() {
	manager := sessions.NewManager(`+store+`)
}`)[0]
	newHandler.Body.List = insertAfter(newHandler.Body.List, 0, manager)
}

func passManagerToHome(newHandler *dst.FuncDecl) {
	for _, call := range methodCalls(newHandler, "router", "HandleFunc") {
		if home, ok := call.Args[1].(*dst.CallExpr); ok && isIdent(home.Fun, "home") {
			home.Args = []dst.Expr{dst.NewIdent("deps"), dst.NewIdent("manager")}
		}
	}
}

func registerLogout(newHandler *dst.FuncDecl) {
	homeRoute := indexOfStmt(newHandler.Body.List, isRoute(`"GET /{$}"`))
	logout := snippetStmts(bootstrapPkg, `package bootstrap

func _() {
	router.HandleFunc("POST /logout", logout(deps, manager))
}`)[0]
	newHandler.Body.List = insertAfter(newHandler.Body.List, homeRoute, logout)
}

func wrapInSessionMiddleware(newHandler *dst.FuncDecl) {
	ret := newHandler.Body.List[len(newHandler.Body.List)-1].(*dst.ReturnStmt)
	ret.Results[0] = snippetExpr(bootstrapPkg, "package bootstrap\n\nfunc _() {\n\tmanager.Middleware(router)\n}")
}

func replaceHomeWithSessionHandlers(handler *dst.File) {
	handlers := snippetFuncs(bootstrapPkg, `package bootstrap

import (
	"fmt"
	"net/http"
	"strconv"

	"example.com/myapp/internal/sessions"
)

func home(deps Dependencies, manager *sessions.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := sessions.FromContext(r.Context())
		visits, _ := strconv.Atoi(session["visits"])
		session["visits"] = strconv.Itoa(visits + 1)

		if err := manager.Save(w, r, session); err != nil {
			deps.Logger.Error("saving session", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Hello, visit %d\n", visits+1)
	}
}

func logout(deps Dependencies, manager *sessions.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := manager.Destroy(w, r); err != nil {
			deps.Logger.Error("destroying session", "error", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}`)
	replaceDecl(handler, funcDecl(handler, "home"), handlers...)
}

func replaceDecl(file *dst.File, old dst.Decl, replacements ...dst.Decl) {
	for i, decl := range file.Decls {
		if decl == old {
			file.Decls = append(file.Decls[:i], append(replacements, file.Decls[i+1:]...)...)
			return
		}
	}
}

// Placeholders are SQL inside Go string literals, which dst sees only as opaque text.
func useSQLitePlaceholders(files goFiles) error {
	dst.Inspect(files[sqlStorePath], func(node dst.Node) bool {
		if lit, ok := node.(*dst.BasicLit); ok && strings.HasPrefix(lit.Value, "`") {
			lit.Value = strings.NewReplacer("$1", "?", "$2", "?").Replace(lit.Value)
		}
		return true
	})
	return nil
}
