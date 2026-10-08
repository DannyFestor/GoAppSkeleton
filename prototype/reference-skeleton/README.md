# PROTOTYPE: reference Skeleton (zero Options)

Throwaway. Answers [Reference Skeleton: hand-write the zero-Option web app](https://github.com/DannyFestor/GoAppSkeleton/issues/6) on the [GoAppSkeleton generator spec map](https://github.com/DannyFestor/GoAppSkeleton/issues/1). It is what the Generator would emit when only the module path is chosen; every Option is later designed as a delta against it.

```sh
make run      # http://localhost:4000
make test
```

## Decisions to react to

| Concern | Prototype choice |
|---|---|
| Layout | `cmd/web` (process entry and composition root), `internal/<slice>` per feature, `internal/httpx` for feature-agnostic HTTP plumbing, `internal/config`, `ui/` for embedded templates and static files. |
| Wiring | `main` builds the process environment; `run(ctx, env) int` is testable; `newHandler` in `cmd/web/handler.go` is the only place slices are constructed and mounted. No globals, no DI container, no interfaces until a second implementation exists. |
| Slice shape | `type Handler struct{deps}`, `New(deps) *Handler`, `Register(mux)`. The slice owns its route patterns. |
| Handler shape | `func(w, r) error`, adapted by `Responder.Handle`. `httpx.StatusError` is shown to the client; any other error is logged and becomes a generic 500. |
| Rendering | `html/template`, every page parsed at boot (broken template fails startup), rendered into a buffer before writing. |
| Config | Environment variables only (`ADDR`, `LOG_LEVEL`), parsed into a typed `Config` with all errors joined. No `.env` loader. |
| Logging | `log/slog` text handler on stdout, level from config, injected (no `slog.SetDefault`). `http.Server.ErrorLog` routed through it. |
| Server | Timeouts set, graceful shutdown on SIGINT/SIGTERM with a 10s drain. Panic recovery middleware. No request logging (that is an Option). |
| 404/405 | Left to `http.ServeMux` (plain text). A catch-all `/` route for a custom 404 page breaks ServeMux's 405 handling, so custom error pages stay an Option. |
| Tests | stdlib `testing` + `httptest`, table-driven. Unit tests per package; one route-table test through the composed handler; `run` tested for bad config and clean shutdown. No services needed. |
| Dependencies | None. |
