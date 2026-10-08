package refpatch

import (
	"github.com/dave/dst"
)

const (
	mainPath     = "cmd/web/main.go"
	handlerPath  = "internal/bootstrap/handler.go"
	mainPkg      = referenceModule + "/cmd/web"
	bootstrapPkg = referenceModule + "/internal/bootstrap"
	sqlStorePath = "internal/sessions/sql_store.go"
)

func addDatabase(choice string) func(goFiles) error {
	return func(files goFiles) error {
		openDatabaseInRun(files[mainPath])
		addDependency(files, "DB *sql.DB", `"database/sql"`, "DB: db")
		pingDatabaseInHealth(files[handlerPath])
		return addOverlay(files, "database/"+choice)
	}
}

func openDatabaseInRun(main *dst.File) {
	run := funcDecl(main, "run")
	opening := snippetStmts(mainPkg, `package main

import (
	"os"

	"example.com/myapp/internal/database"
)

func _() {
	db, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
}`)
	opening[len(opening)-1].Decorations().After = dst.EmptyLine
	run.Body.List = insertStmts(run.Body.List, 0, opening...)
}

func pingDatabaseInHealth(handler *dst.File) {
	closure := returnedFuncLit(funcDecl(handler, "health"))
	ping := snippetStmts(bootstrapPkg, `package bootstrap

import "net/http"

func _() {
	if err := deps.DB.PingContext(r.Context()); err != nil {
		deps.Logger.Error("health check", "error", err)
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
}`)
	closure.Body.List = insertStmts(closure.Body.List, 0, ping...)
}

// addDependency threads one value from main.run into bootstrap.Dependencies: a struct field plus its initialiser.
func addDependency(files goFiles, field, fieldImport, initialiser string) {
	deps := structType(files[handlerPath], "Dependencies")
	deps.Fields.List = append(deps.Fields.List, snippetFields(bootstrapPkg, "package bootstrap\n\nimport "+fieldImport+"\n\ntype _ struct {\n"+field+"\n}")...)

	literal := compositeLit(funcDecl(files[mainPath], "run"), "Dependencies")
	element := snippetCompositeElement(initialiser)
	element.Decorations().Before = dst.NewLine
	element.Decorations().After = dst.NewLine
	literal.Elts = append(literal.Elts, element)
}

func snippetCompositeElement(initialiser string) dst.Expr {
	literal := snippetExpr(mainPkg, "package main\n\nimport \"os\"\n\nfunc _() {\n\tT{"+initialiser+"}\n}").(*dst.CompositeLit)
	return literal.Elts[0]
}
