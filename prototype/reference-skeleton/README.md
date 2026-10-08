# PROTOTYPE: reference Skeleton (zero Options)

Throwaway. Answers [Reference Skeleton: hand-write the zero-Option web app](https://github.com/DannyFestor/GoAppSkeleton/issues/6) on the [GoAppSkeleton generator spec map](https://github.com/DannyFestor/GoAppSkeleton/issues/1). It is what the Generator would emit when only the module path is chosen; every Option is later designed as a delta against it.

```sh
make run      # http://localhost:4000
make test     # unit, feature, e2e
```

## Decisions to react to

| Concern | Prototype choice |
|---|---|
| Layout | Tests in `tests/` (Laravel convention). `cmd/web` (process entry only), `internal/bootstrap` (composition root, importable by feature tests), `internal/<slice>` per feature, `internal/httpx` for feature-agnostic HTTP plumbing, `internal/config`, `ui/` for embedded templates, `public/` for static assets on disk. |
| Wiring | Dependency injection everywhere through exported constructors. `main` builds the process environment; `run(ctx, env) int` is testable; `bootstrap.NewHandler` is the only place slices are constructed and mounted. No globals, no DI container. |
| Interfaces | Declared by the consumer, next to the code that uses them (`home.responder`, `httpx.pageRenderer`, `httpx.errorResponder`), never by the provider. Duplicating a small interface across consumers is fine; a provider-owned interface invites import cycles. |
| Slice shape | `type Handler struct{deps}`, `New(deps) *Handler`, `Register(mux)`. The slice owns its route patterns. |
| Handler shape | Plain `http.HandlerFunc`, as close to the stdlib as possible. A failing handler calls `responder.Error(w, r, err)`: `httpx.StatusError` is shown to the client; any other error is logged and becomes a generic 500. |
| Middleware | `httpx.Middleware` is `func(http.Handler) http.Handler`; `httpx.Chain(handler, a, b)` applies them with the first listed outermost, replacing nested `a(b(handler))`. |
| Rendering | Web application type only (a JSON API Choice replaces it). `html/template`, every page parsed at boot (broken template fails startup), rendered into a buffer before writing. |
| Config | Environment variables only (`ADDR`, `LOG_LEVEL`, `STATIC_DIR`), parsed into a typed `Config` with all errors joined. This is the default Choice of a configuration-source Option (flags, env, `.env`, cobra). Fixed default port `:4000`. |
| Static files | Templates are embedded; CSS, JS and images live in `public/` on disk, served at `/static/` from `STATIC_DIR`. |
| Logging | `log/slog` text handler on stdout, level from config, injected wherever needed; never `slog.SetDefault`, a global would set a bad precedent. `http.Server.ErrorLog` routed through it. |
| Server | Timeouts set, graceful shutdown on SIGINT/SIGTERM with a 10s drain. Panic recovery middleware. No request logging (that is an Option). |
| 404/405 | Left to `http.ServeMux` (plain text). A catch-all `/` route for a custom 404 page breaks ServeMux's 405 handling, so custom error pages stay an Option. |
| Tests | stdlib `testing` + `httptest`, table-driven. Three tiers: unit tests beside their implementation, using fakes of the consumer-side interfaces; `tests/feature/` (build tag `feature`) drives `bootstrap.NewHandler` in process; `tests/e2e/` (build tag `e2e`) builds the binary, waits for `/healthz`, sends SIGTERM and expects a clean exit. No services needed. |
| Dependencies | None. |
