# Prior art: autostrada.dev and go-blueprint

Research for issue #2 (map: #1). Fetched and verified 2026-10-08.

**Prior art is a guide only.** Nothing here is copied into the Generator. This document records what two existing tools offer, how one of them is built, and what their histories say about composing Templates across several Options, so that the catalogue and the engine prototype tickets can cite one place.

Vocabulary follows `GLOSSARY.md`: Generator, Option, Choice, Configuration, Skeleton, Template.

## 1. autostrada.dev

Closed-source, server-side web form that emits a ZIP. Author: Alex Edwards. Counter on the home page reads "23,469 projects created" ([autostrada.dev](https://autostrada.dev)).

### 1.1 Options and Choices

Source: the generator form at [autostrada.dev](https://autostrada.dev), verified 2026-10-08.

| Step | Option | Choices | Tier |
|---|---|---|---|
| 1 | Application type | Traditional web application (SSR HTML), JSON API | Free |
| 2 | Database | None, SQLite, PostgreSQL, MySQL | Free |
| 2 | SQL migrations helpers | checkbox | Free |
| 2 | Router | http.ServeMux, Flow, Chi, Gorilla Mux, HttpRouter | Free |
| 2 | Configuration source | Command-line flags, Environment variables | Free |
| 2 | Log format | Plaintext, JSON, Colorized plaintext | Free |
| 2 | Request logging middleware | checkbox | Free |
| 2 | Basic auth | checkbox | Free |
| 2 | Git integration | checkbox (`.gitignore` plus version from VCS build info) | Free |
| 2 | Secure cookies | checkbox (signed and encrypted cookie helpers) | Free |
| 2 | Server-side sessions | checkbox; web only; requires a database | Free |
| 2 | Module path | text | Free |
| 3 | Automatic HTTPS | checkbox (autocert, HTTP to HTTPS redirect) | Plus |
| 3 | Custom error pages | checkbox (400/404/500; web only) | Plus |
| 3 | Emails | checkbox (templates plus SMTP) | Plus |
| 3 | Error notifications | checkbox (runtime errors emailed to an admin) | Plus |
| 3 | Tests | checkbox, marked "New" | Plus |
| 3 | User accounts and authentication | web: signup, login, logout, password reset, auth middleware, CSRF; API: email/password with JWT | Plus |

Always on, no longer Options: Admin Makefile and live reloading (Air). The 26 April 2026 changelog entry "Feature reorganization" removed those two Options, replaced "Automatic versioning" and "Gitignore" with "Git integration", and moved Emails and Error notifications behind Plus ([changelog](https://autostrada.dev/changelog)).

Cross-Option constraints visible in the form: Server-side sessions, Custom error pages and the full user-accounts variant exist only for the web application type; sessions need a database. The JSON API docs navigation ([docs/api](https://autostrada.dev/docs/api)) has no sessions, custom error pages, or user accounts pages, while the web docs navigation ([docs/web](https://autostrada.dev/docs/web)) has all three.

### 1.2 Roadmap

Source: [autostrada.dev/roadmap](https://autostrada.dev/roadmap), which opens with "this roadmap isn't set in stone".

Coming soon: Error wrapping; Code comments (optional comments on all generated code).

Possible future, unordered: Email confirmations; IP rate limiting (`x/time/rate`); shareable, reproducible application configurations; Request IDs; RPC service Skeletons; CLI tool Skeletons; PASETO instead of JWT; CORS middleware for APIs; `net/http/pprof`; metrics (memory, goroutines, DB pool); I18N/L10N; pagination helpers; WebSockets and server-sent events; OAuth/OpenID Connect; Redis; email providers (SendGrid, Postmark, SES); CSS frameworks (Tailwind, Bootstrap); HTML-over-the-wire (HTMX, Hotwire, Unpoly); analytics; GitHub Actions CI; Sentry; Docker and docker-compose; deployment tooling (DigitalOcean, AWS); ntfy.sh notifications.

Note for the catalogue: "shareable application configurations" is a Configuration-as-artifact idea; autostrada does not have it yet.

### 1.3 Shape of the Skeleton

Sources: [docs/web/getting-started](https://autostrada.dev/docs/web/getting-started), [docs/web/admin-makefile](https://autostrada.dev/docs/web/admin-makefile), [docs/web/sql-migrations](https://autostrada.dev/docs/web/sql-migrations), [docs/web/server-side-sessions](https://autostrada.dev/docs/web/server-side-sessions), changelog entries, and the third-party example [adamjsturge/autostrada-example](https://github.com/adamjsturge/autostrada-example) (Go 1.21 era; file list only, treat as dated).

- `cmd/web` or `cmd/api`: `main.go` (config struct, `application` struct, `run()`), `routes.go`, `handlers.go`, `middleware.go`, `helpers.go`, `errors.go`, `context.go`, `server.go`.
- `assets/`: `efs.go` (`embed.FS`), `migrations/*.up.sql` and `*.down.sql`, `emails/*.tmpl`, `static/`, `templates/`. Contents vary by Configuration.
- `internal/`: `cookies`, `database` (`New()`, `MigrateUp()`, sqlx), `funcs`, `password` (bcrypt), `request` (`DecodePostForm`, `DecodeQueryString`, `DecodeJSONObject`/`Array`), `response` (`HTMLRenderer` since 5 June 2026, JSON), `smtp`, `validator`, `version`, `tokens`.
- `Makefile`: `help`, `audit` (vet, staticcheck, govulncheck, gofmt, `go mod tidy -diff`), `test`, `test/cover`, `upgradeable`, `tidy` (runs `go fix` since 20 April 2026), `build`, `run`, `run/live` (Air), `migrations/{new,up,down,goto,force,version}` via the golang-migrate CLI.
- Random default port per generated codebase (changelog 3 April 2025).

Dependencies named in docs or changelog: `jmoiron/sqlx`; `mattn/go-sqlite3`, `lib/pq`, `go-sql-driver/mysql`; `golang-migrate/migrate/v4` (migrations embedded, `--db-automigrate` flag; `database.New()` stopped auto-migrating 22 October 2025); `alexedwards/scs` with `sqlite3store`/`postgresstore`/`mysqlstore` (replaced gorilla/sessions 8 July 2025; verified on the sessions docs page); `alexedwards/flow`, `go-chi/chi`, `gorilla/mux`, `julienschmidt/httprouter`; `justinas/nosurf` (CSRF); `lmittmann/tint` (colorized logs); `log/slog`; `wneessen/go-mail`.

Go version: not stated, but the output tracks the newest Go release. Changelog: 3 October 2026 switched to Go 1.27 `encoding/json/v2` and `url.Clone()`; 20 April 2026 used `sync.WaitGroup.Go()` (Go 1.25); the validator moved from `x/exp/constraints` to `cmp`.

### 1.4 Pricing and licence

Free tier: all Step 1 and Step 2 Options. Autostrada Plus: "$29 USD", "Lifetime account, no recurring payments" ([signup](https://autostrada.dev/signup)); 30-day refund ([guarantee](https://autostrada.dev/guarantee)). Generated code is MIT with no attribution requirement ([terms](https://autostrada.dev/terms-of-service)).

### 1.5 Unverified

- Generator internals (template engine, how Options compose). No public repository; nothing under github.com/alexedwards or via `gh search`.
- JWT library for the API user-accounts Choice. The API auth docs page is not in the public navigation (returns 404 at the guessed URL). `pascaldekloe/jwt` is inferred from the 2024 third-party example only.
- sqlx pool settings (25/25, 5m idle, 2h lifetime, 3s timeout) come from the 2024 example, not current docs.

## 2. go-blueprint (Melkeydev/go-blueprint)

Open-source CLI, MIT, 8,957 stars. Latest commit on `main` is 81f56f8 (2025-07-19); latest release v0.10.11 (2025-07-10). Not archived, but `main` has had no commits in the 15 months since, and 15 PRs are open with no merges, the newest being #441 "Add huma framework" (2026-07-17), #440 "swagger advanced feature" (2026-05-13), #439 "Next.js feature option" (2026-04-23). A commenter on issue #434 wrote "I think the project is dead" (2025-12-26). Treat as dormant. (Sources: GitHub API for the repo, `commits/main`, `releases/latest`, `gh pr list`, issue #434 comments.)

### 2.1 Options and Choices

Source: `cmd/flags/*.go` on `main` at 81f56f8; the module-name regex is in `cmd/utils/utils.go` line 155.

| Option | Flag | Choices |
|---|---|---|
| Project name / module path | `--name` | text matching `^[a-zA-Z0-9_-]+(?:[\/.][a-zA-Z0-9_-]+)*$` |
| Framework | `--framework` | `chi`, `gin`, `fiber`, `gorilla/mux`, `httprouter`, `standard-library`, `echo` |
| Database driver | `--driver` | `mysql`, `postgres`, `sqlite`, `mongo`, `redis`, `scylla`, `none` |
| Git | `--git` | `commit`, `stage`, `skip` |
| Advanced features | `--advanced --feature` (repeatable) | `react`, `htmx`, `githubaction`, `websocket`, `tailwind`, `docker` |

Any flag not supplied triggers a Bubble Tea TUI step for that Option; after generation the CLI prints the equivalent non-interactive command.

Cross-Option constraints are imperative mutations inside `CreateMainFile` in `cmd/program/program.go` (lines 410 to 423, 929), not a declared rule set:

- `react` forces `htmx=false`, then `tailwind=false` for the Go side (Tailwind is installed into the Vite project instead).
- `tailwind` without `react` forces `htmx=true`.
- `sqlite` skips `database_test.go` and the DB compose service.
- `docker` with `none` or `sqlite` yields a generic compose file; with any other driver it adds the app service.
- `websocket` picks `gofiber/contrib/websocket` for Fiber and `coder/websocket` for everyone else (line 965).
- `scylla` runs `go mod edit -replace github.com/gocql/gocql=github.com/scylladb/gocql@v1.14.4` (lines 107 to 108, 358).
- `--git commit` or `stage` panics if `user.name` or `user.email` is unset (lines 263 to 265, 702 to 704).

Nothing rejects a conflicting Configuration; issue #311 "CLI Flow Control" (open) asks for validation.

### 2.2 Shape of the Skeleton

Default: `cmd/api/main.go`, `internal/server/{server.go,routes.go,routes_test.go}`, `Makefile`, `README.md`, `.env` (`PORT=8080`, `APP_ENV=local`), `.gitignore`, `.air.toml`, `go.mod`/`go.sum`; `godotenv` is always a dependency.

Per Choice:

- Any database: `internal/database/database.go` (`Service` interface with `Health`/`Close`, singleton `New`), `database_test.go` using testcontainers (not for sqlite), `docker-compose.yml`, DB env vars, a `/health` route, and `itest`/`docker-run`/`docker-down` Make targets.
- `htmx`: `cmd/web/{hello.templ,base.templ,efs.go,hello.go}`, `cmd/web/assets/js/htmx.min.js`, `tailwind.config.js` (even without tailwind), the `templ` dependency, routes spliced into `routes.go`, `templ-install`/`templ generate` Make targets, `.air.toml` watching `.templ`.
- `tailwind`: `input.css`, `output.css`, a `tailwind-install` target that curls the standalone CLI.
- `react`: runs `npm create vite@latest frontend --template react-ts`, overwrites `App.tsx`, writes `frontend/.env`; with tailwind, `npm install tailwindcss@^4 @tailwindcss/vite`; `make run` backgrounds the Go server and runs `npm run dev`.
- `websocket`: injects a `/websocket` handler into `routes.go`.
- `docker`: multi-stage `Dockerfile` (`templ generate`, tailwind curl, `CGO_ENABLED=1` for sqlite, node stages for react); `.env` DB host switches to the compose service name.
- `githubaction`: `.github/workflows/{release.yml,go-test.yml}`, `.goreleaser.yml`.

### 2.3 Engine structure

Source: `cmd/program/program.go` (1,000 lines), `cmd/template/framework/files/`, `cmd/utils/utils.go`.

Three mechanisms compose Templates across Options:

1. **Strategy objects per framework (file selection).** Interfaces at `program.go` lines 68 to 92:

   ```go
   type Templater interface {
       Main() []byte; Server() []byte; Routes() []byte; TestHandler() []byte
       HtmxTemplRoutes() []byte; HtmxTemplImports() []byte; WebsocketImports() []byte
   }
   type DBDriverTemplater interface { Service() []byte; Env() []byte; Tests() []byte }
   type DockerTemplater interface { Docker() []byte }
   type WorkflowTemplater interface { Releaser() []byte; Test() []byte; ReleaserConfig() []byte }
   ```

   One empty struct per framework returns `go:embed`-ed `.tmpl` bytes; `FrameworkMap`, `DBDriverMap` and `DockerMap` pick the strategy. Template files on `main`: routes has one file per framework (7); server has `fiber.go.tmpl` and `standard_library.go.tmpl` (six frameworks share the latter); main has `fiber_main.go.tmpl` and `main.go.tmpl`; tests has `default`, `echo`, `fiber`, `gin`.

2. **`text/template` conditionals inside each file (the second axis).** Database, websocket, docker and OS are `{{ if }}` blocks inside the per-framework Templates. Each routes Template carries around ten conditionals; the single Makefile Template has nested conditionals since #272 merged two Makefiles into one. Across the Template tree there are roughly 21 websocket, 20 `DBDriver == "none"` and 19 docker conditionals, so one feature change touches every framework file.

3. **Pre-rendered string fragments (the third axis).** htmx and websocket imports and routes are rendered first, stored as strings on `p.AdvancedTemplates`, then printed into the routes Template. Import correctness depends on running `gofmt -s -w .` afterwards. The websocket handler body is duplicated in all seven routes Templates. `.env` is assembled with `bytes.Join`. `CreateFileWithInjection` (line 748) is a `switch` over string method names; `CreateMainFile` (line 245) is about 500 lines of procedural steps.

Shell-outs during generation (`utils.go`, `program.go`): `go mod init`, `go get -u` per dependency, `go mod edit -replace`, `go mod tidy`, `gofmt -s -w .`, `git config --get`, `git init`, `git add .`, `git commit`, `npm --version`, `npm create vite@latest`, `npm install`. Generation therefore needs a Go toolchain, network access, git (panic without it, #360) and npm for react. Because of `go get -u`, dependency versions differ per run (#379).

### 2.4 Testing of the engine

Workflows in `.github/workflows/`: `ci.yml` lints and tests the CLI itself (no Template unit tests). `generate-linter-core.yml` runs a matrix of 7 frameworks x 7 drivers x 3 git modes = 147 jobs, each generating a Skeleton then running `golangci-lint run` (verified matrix in the file). `generate-linter-advanced.yml` runs 7 frameworks x postgres x commit x one feature at a time = 42 jobs. `testcontainers.yml` runs `make itest` for fiber with mysql, postgres, mongo, redis, scylla. Feature combinations are never tested together, and no workflow runs `go test` on a generated server. `update-htmx-version.yml` is a weekly bot. Also present: `docs.yml`, `npm-publish.yml`, `release.yml`.

### 2.5 Recorded pain points

From the issue tracker (states verified 2026-10-08):

- #311 open: CLI Flow Control. Hidden dependencies and conflicts between Options; nothing validates a Configuration.
- #331 closed: community vote to remove HttpRouter or Gorilla Mux because each framework multiplies maintenance.
- #272 closed: merged two Makefile Templates into one heavily conditional file.
- #312, #334 open; #292, #235 closed: multiple databases at once, repeatedly requested and dismissed. The engine assumes one Choice per Option.
- #350 closed: the React frontend hard-coded port 8080, a coupling across Templates.
- #416 open: `.env` committed by `--git commit`.
- #421 open; #394, #382, #387, #404, #371 closed: React plus Tailwind npm `ERESOLVE` errors and version churn from live `npm`/`go get -u` fetches.
- #434 open: Postgres 18 data-directory change broke the compose Template; #361 Air crash; #289 sqlite compose failure.
- #17 open: no ORMs, by design. #427, #305 (GORM, ent) closed. #327 sqlc open.
- #253 open: CRUD example. #268 open: `html/template` alternative to templ.
- #379 closed: Echo generation failed because of unpinned `go get -u`.
- #360 closed: panic when git is absent.

### 2.6 Docs and visualiser

Docs are MkDocs Material (Installation, Blueprint UI, Project init/Makefile/Air, Core: Frameworks and DB Drivers, Advanced flag pages, Testing endpoints). Static ASCII trees are partly aspirational. The visualiser is a separate app, Blueprint UI at go-blueprint.dev (repo `briancbarrow/go-blueprint-htmx`, Go plus templ plus htmx, last push 2024-06), a hand-maintained duplicate of the layout rules that drifted from the CLI.

## 3. Lessons for the Generator's engine

Drawn from section 2; prior art is a guide only.

1. **Declare Option constraints as data, not as mutations.** go-blueprint's rules live as `if` statements in a 500-line function, so neither the TUI nor the web visualiser can show them, and #311 stays open. A Configuration should be validated against declared rules before any Template is rendered.
2. **Avoid conditionals that fan out across every per-Choice file.** Twenty-ish `{{ if websocket }}` blocks spread over seven routes Templates means each new Option edits every framework Template. Prefer composing a file from parts owned by each Choice, or keep per-Choice fragments in one place.
3. **Do not pre-render strings and splice them into other Templates.** Relying on `gofmt` to fix imports after string splicing is fragile and hides errors until the generated code is compiled.
4. **Make generation hermetic.** Shell-outs to `go get -u`, `npm create` and `git` make output non-reproducible (#379, #421, #360) and require a toolchain on the host. Pin dependency versions inside the Generator and write `go.mod`/`go.sum` directly, or at minimum pin exact versions.
5. **Test Configurations, not just axes.** 147 single-axis lint jobs never caught feature-combination bugs. The test matrix should include combinations of Choices and should compile and `go test` the Skeleton.
6. **One source of truth for the catalogue.** The separate visualiser drifted. Docs, CLI, and web UI should read the same Option and Choice definitions.
7. **Multi-select Options need a decision early.** go-blueprint's single-Choice-per-Option assumption blocked multiple-database requests four times.
8. **Dependency currency has a cost either way.** autostrada tracks the newest Go release within days; go-blueprint's live fetches broke on upstream releases. Either way the Generator needs a documented policy.

## 4. Inertia for Go

Verified via GitHub API 2026-10-08. Neither autostrada nor go-blueprint offers an Inertia Choice.

| Adapter | Stars | Licence | Latest release | Notes |
|---|---|---|---|---|
| [romsar/gonertia](https://github.com/romsar/gonertia) | 266 | NOASSERTION per GitHub (check the LICENSE file before relying on it) | v3.0.0, 2026-04-11 | `net/http`, zero deps, SSR, asset versioning, test helpers |
| [petaki/inertia-go](https://github.com/petaki/inertia-go) | 328 | MIT | v1.16.0, 2026-09-02 | community forks exist for Fiber, Vite, Echo |
| [mayahiro/go-inertia](https://github.com/mayahiro/go-inertia) | 0 | MIT | `adapters/echo/v0.5.0`, 2026-08-22; root module v0.3.0 (May 2026, per earlier report) | Inertia 3.x protocol, stdlib, Go 1.25+, deferred and merge props, Echo v5 adapter |
| [goravel/inertia](https://github.com/goravel/inertia) | 8 | MIT | pushed 2026-09-09 | Goravel-framework specific |

## 5. Other comparables

- `gonew` ([go.dev/blog/gonew](https://go.dev/blog/gonew)): copies a module and rewrites the module path; no Options.
- [create-go-app/cli](https://github.com/create-go-app/cli): 2,769 stars, Apache-2.0, pushed 2026-02-10; clones Template repositories rather than rendering.
- [allaboutapps/go-starter](https://github.com/allaboutapps/go-starter): 623 stars, MIT, pushed 2026-08-20; a single Template repo with `make set-module-name`.
- Goyave: install script clones a Template repo.
- Buffalo: [gobuffalo/cli](https://github.com/gobuffalo/cli) is archived (last push 2024-03-19).

## 6. One-line comparison

go-blueprint has breadth of infrastructure Choices (Gin, Fiber, Echo, Gorilla, HttpRouter; Mongo, Redis, Scylla; testcontainers; Docker and compose; GoReleaser and Actions; htmx plus templ; Tailwind; React via Vite; WebSocket; TUI with printable flag equivalents; npm and Homebrew distribution) but no application layer, while autostrada has the application layer (migrations, configuration source, log formats, request logging, basic auth, sessions, cookies, HTTPS, error pages, email, error notifications, user accounts, SSR versus API split, generated tests) but no frontend tooling, Docker, CI, NoSQL, or open engine.

## 7. Caveats

- go-blueprint was inspected from source and the GitHub API, not run. Conditional counts are from an earlier grep and are approximate.
- autostrada's engine is closed; everything about its internals is unknown.
- Not every go-blueprint issue was read; the pain-point list is the set found by searching for the themes above.
