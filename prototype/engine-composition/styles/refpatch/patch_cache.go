package refpatch

import (
	"github.com/dave/dst"
)

func addCache(files goFiles) error {
	openCacheInRun(files[mainPath])
	addDependency(files, "Cache *redis.Client", `"github.com/redis/go-redis/v9"`, "Cache: cacheClient")
	pingCacheInHealth(files[handlerPath])
	return addOverlay(files, "cache/redis")
}

// Anchored on the Dependencies statement rather than index 0, so the cache opens after any database the patches added.
func openCacheInRun(main *dst.File) {
	run := funcDecl(main, "run")
	opening := snippetStmts(mainPkg, `package main

import (
	"os"

	"example.com/myapp/internal/cache"
)

func _() {
	cacheClient, err := cache.Open(ctx, os.Getenv("REDIS_URL"))
	if err != nil {
		return err
	}
	defer cacheClient.Close()
}`)
	opening[len(opening)-1].Decorations().After = dst.EmptyLine
	run.Body.List = insertStmts(run.Body.List, indexOfStmt(run.Body.List, assigns("deps")), opening...)
}

// Inserted before the closing "ok" write, so it follows a database ping when there is one.
func pingCacheInHealth(handler *dst.File) {
	closure := returnedFuncLit(funcDecl(handler, "health"))
	ping := snippetStmts(bootstrapPkg, `package bootstrap

import "net/http"

func _() {
	if err := deps.Cache.Ping(r.Context()).Err(); err != nil {
		deps.Logger.Error("health check", "error", err)
		http.Error(w, "cache unavailable", http.StatusServiceUnavailable)
		return
	}
}`)
	closure.Body.List = insertStmts(closure.Body.List, len(closure.Body.List)-1, ping...)
}

func assigns(name string) func(dst.Stmt) bool {
	return func(stmt dst.Stmt) bool {
		assign, ok := stmt.(*dst.AssignStmt)
		return ok && len(assign.Lhs) == 1 && isIdent(assign.Lhs[0], name)
	}
}
