# Mechanisms for generating Go source from Templates

Research ticket: [#3](https://github.com/DannyFestor/GoAppSkeleton/issues/3), part of the map [#1](https://github.com/DannyFestor/GoAppSkeleton/issues/1). Feeds the engine composition prototype [#7](https://github.com/DannyFestor/GoAppSkeleton/issues/7).

## Question

Which mechanisms are viable for an engine that renders a Skeleton's Go, SQL, YAML and Makefile sources from Templates that vary along several Options at once, with no shell-outs at generation time?

## Constraints taken from the map

- The engine is a pure function from Configuration to files. No subprocess at generation time; `go/format` in-process is allowed.
- CLI and web front ends must produce byte-identical output for the same Configuration, so output must not depend on the host's module cache, GOPATH or installed tools.
- Output file kinds: Go, SQL, YAML, Makefile, plus dotfiles (`.gitignore`, `.air.toml`), Dockerfile, `go.mod`/`go.sum`.
- Options interact: one file (typically `main.go`, `server.go`, `Makefile`) is touched by router, database and session-store Choices at once.

## Method

Primary sources: Go package docs and the Go source tree for `text/template`, `go/format`, `go/ast`, `embed`, `golang.org/x/tools/imports`; templ docs and source; the go-blueprint repository at commit `81f56f8` (2025-07-20); dst and jennifer source; source of sqlc, oapi-codegen, ent, gqlgen, go-swagger, goa, kubebuilder, genny, gowebly, gonew; cookiecutter, copier, Yeoman, Nx and Rails documentation. Every claim below links its source.

Local experiments ran on Go 1.27.1 (darwin/arm64) with `golang.org/x/tools` v0.51.0, `dave/dst` v0.28.0, `dave/jennifer` v1.7.1 and templ v0.3.1070. "Isolated" means the built binary ran with empty `GOPATH`/`GOMODCACHE`/`HOME` and no `go` on `PATH`.

## Building blocks: what the standard library guarantees

### `text/template`

- Text outside actions is copied verbatim, so Makefile tabs and YAML indentation survive. Trim markers `{{-`/`-}}` eat all adjacent whitespace including tabs and newlines, which is the usual way to lose a Makefile recipe tab. [doc.go, Text and spaces](https://pkg.go.dev/text/template#hdr-Text_and_spaces)
- `{{template "x"}}` without a pipeline passes nil as dot; always write `{{template "x" .}}`. Inside `range`/`with`, `$` is the root. [Actions](https://pkg.go.dev/text/template#hdr-Actions)
- `Option("missingkey=error")` only governs map lookups; a missing struct field always errors. Verified locally: `map has no entry for key "Nope"` and `can't evaluate field Nope in type struct {}`. [option.go](https://github.com/golang/go/blob/master/src/text/template/option.go)
- Redefinition rules, from source:
  - Within one `Parse` call, two non-empty definitions of the same name are a hard error (`multiple definition of template`). [parse.go `(*Tree).add`](https://github.com/golang/go/blob/master/src/text/template/parse/parse.go#L284)
  - Across successive `Parse` calls, the last non-empty definition wins silently. "Templates can be redefined in successive calls to Parse. A template definition with a body containing only white space and comments is considered empty and will not replace an existing template's body." [template.go `Parse`](https://github.com/golang/go/blob/master/src/text/template/template.go#L197), [`associate`](https://github.com/golang/go/blob/master/src/text/template/template.go#L227)
  - Verified locally: parsing an overlay `{{define "imports"}}...{{end}}` after a base replaced the base's block, even after `Execute` had run (the current source has no "before first Execute" check; concurrent Parse and Execute would race).
  - Pitfall verified locally: an overlay cannot blank out a block with `{{define "mid"}}{{end}}` because that body counts as empty. `{{define "mid"}}{{""}}{{end}}` works, since an action is not whitespace.
- `Clone` is the documented pattern for "common base plus variant definitions": clone, then parse the variant set into the clone. [template.go `Clone`](https://github.com/golang/go/blob/master/src/text/template/template.go#L81)
- `ParseFS` names templates by file base name; "the last one mentioned will be the one that results" when two directories hold the same base name. A per-Choice directory layout therefore needs the engine to walk the FS and call `t.New(relpath).Parse(...)`, as oapi-codegen does. [helper.go](https://github.com/golang/go/blob/master/src/text/template/helper.go)
- `html/template` must not be used for Go, SQL or YAML: its contextual autoescaper rewrites actions at parse time and escapes `<`, `>`, `&`, `'`, `"` according to the HTML/JS/CSS/URL context it infers. [html/template](https://pkg.go.dev/html/template)

### `go/format`

- `format.Source` is gofmt without `-s`, in-process, no subprocess. It accepts a whole file, a declaration list or a statement list (it retries with a synthetic `package p;` and `func _() {` wrapper). Imports are sorted only for whole files. [format.go](https://github.com/golang/go/blob/master/src/go/format/format.go), [internal.go](https://github.com/golang/go/blob/master/src/go/format/internal.go)
- Verified locally: it sorts an import block and normalises whitespace but does not remove an unused `database/sql` import. Unused imports are a compile error in the Skeleton, so something else must guarantee the import list.
- Package doc warning: "formatting of Go source code changes over time, so tools relying on consistent formatting should execute a specific version of the gofmt binary". For this Generator the Go version is pinned by the Generator's own build, which is enough for byte-identical CLI and web output. [go/format](https://pkg.go.dev/go/format)

### `golang.org/x/tools/imports` (goimports as a library)

- `imports.Process` with the default options spawns `go env -json ...` on every call through `ProcessEnv.init()`, before it even looks for missing imports; resolving third-party packages then needs `go list` or a populated module cache. [fix.go `init`](https://github.com/golang/tools/blob/master/internal/imports/fix.go#L1092), [forward.go](https://github.com/golang/tools/blob/master/imports/forward.go#L44)
- Verified locally. With `go` on `PATH`: 300 ms, and `chi.NewRouter()` was resolved to `github.com/go-chi/chi/v5` only because that module happened to be in the local module cache. Isolated: `err: go command required, not found: exec: "go": executable file not found in $PATH`.
- `Options{FormatOnly: true}` skips resolution entirely: pure parse, merge, sort into stdlib/third-party/local groups, print. Verified isolated: works, 0.6 ms, but like `format.Source` it leaves unused imports in place. [imports.go](https://github.com/golang/tools/blob/master/internal/imports/imports.go#L62)
- Conclusion: `imports.Process` in add/remove mode is excluded by the no-shell-out rule and by the byte-identical rule (its result depends on the host). `FormatOnly` is an acceptable `format.Source` substitute that also groups imports goimports-style.

### `go/ast` for splicing

- Comments are free-floating; the printer places them by byte position. Moving or inserting nodes misplaces them. Go issue #20744 (Griesemer, open since 2017): "a major design flaw". [go/ast File](https://pkg.go.dev/go/ast#File), [golang/go#20744](https://github.com/golang/go/issues/20744)
- `astutil.AddImport`, `DeleteImport`, `UsesImport` and `Apply` cover import editing and traversal but not comment migration. [astutil](https://pkg.go.dev/golang.org/x/tools/go/ast/astutil)

### `embed`

- A directory pattern excludes files and directories whose names start with `.` or `_`; `all:` includes them (Go 1.18+). A Skeleton Template tree with `.gitignore`, `.air.toml` or `_test` directories needs `//go:embed all:templates`, or neutral names renamed on write. [embed](https://pkg.go.dev/embed), [Go 1.18 notes](https://go.dev/doc/go1.18#embed)
- Patterns cannot reach outside the package directory, cannot follow symlinks, stop at nested `go.mod`, and ignore empty directories. An empty `migrations/` in the Skeleton must be created by the engine, not embedded. [cmd/go `resolveEmbed`](https://github.com/golang/go/blob/master/src/cmd/go/internal/load/pkg.go)
- `embed.FS` implements `fs.FS`; `fs.Sub` and `fs.WalkDir` (lexical order, deterministic) work on it, and `text/template.ParseFS` accepts it. [embed.FS](https://pkg.go.dev/embed#FS)

## Candidates

### A. `text/template`, whole-file Templates with conditionals

How multi-axis variation is expressed: one Template per output file, every Option visible as `{{if .Options.Websocket}}`, `{{if ne .DB "none"}}`, `{{if and .Tailwind (not .React)}}`. This is go-blueprint's engine and go-micro's.

Evidence from go-blueprint (commit `81f56f8`):

- 72 `.tmpl` files, 78 `//go:embed` directives each binding one file to a `[]byte` variable, 39 Go files of glue. [cmd/template](https://github.com/Melkeydev/go-blueprint/tree/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/template)
- The routes Template carries the same five conditionals copy-pasted across all 7 framework variants. [chi.go.tmpl](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/template/framework/files/routes/chi.go.tmpl#L3-L41)
- The Makefile Template has 16 `{{if}}`s, the Dockerfile 7, including a shell `&& \` continuation emitted conditionally so a `RUN` line joins correctly. [makefile.tmpl](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/template/framework/files/makefile.tmpl#L50-L54), [dockerfile.tmpl](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/template/advanced/files/docker/dockerfile.tmpl#L1-L24)
- Where conditionals ran out, the engine renders fragments to strings and pastes them into two placeholder fields (`TemplateImports`, `TemplateRoutes`) inside the `import (...)` block and the route-registration body; the fragment and host silently share identifiers (`r`, `s.App`, `web`). [CreateHtmxTemplates](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/program/program.go#L932-L962), [CreateWebsocketImports](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/program/program.go#L964-L992)
- Option implications (tailwind implies htmx, react excludes both) are encoded four times: imperative flag mutation in `CreateMainFile`, prose in `steps.go`, `(not .AdvancedOptions.react)` guards in the Makefile Template, and again in the web UI's handler. [program.go L415-428](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/program/program.go#L415-L428), [go-blueprint-htmx update_structure.go](https://github.com/briancbarrow/go-blueprint-htmx/blob/dcdf7e5ba27792a068b3c97cbddbb1bf4dfe427d/cmd/web/update_structure.go)
- Formatting is a subprocess: `go mod tidy` then `gofmt -s -w .` at the end, added in PR #7 so "whitespace problems [are] a non-issue". The Templates depend on it: two-space conditionals inside tab-indented Go, stray blank lines, fragments with mixed indentation. `goimports` is never run, so imports and their uses must sit inside the same `{{if}}`. [program.go L683-695](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/program/program.go#L683-L695), [PR #7](https://github.com/Melkeydev/go-blueprint/pull/7)
- Other shell-outs: `go mod init`, one `go get -u` per package (fetching `@latest` at generation time), `go mod edit -replace`, `git`, `npm create vite@latest`. The react+tailwind bug cluster (#382, #387, #394, #421) comes from upstream versions drifting under those `@latest` fetches. [utils.go](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/utils/utils.go#L62-L133), [#421](https://github.com/Melkeydev/go-blueprint/issues/421)
- CI generates and lints 7 frameworks x 7 drivers, and 7 frameworks x one advanced feature at a time; no feature combination is ever tested. Generated projects did not compile and CI passed until PR #196 added a build step. [generate-linter-advanced.yml](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/.github/workflows/generate-linter-advanced.yml#L8-L49), [PR #196](https://github.com/Melkeydev/go-blueprint/pull/196)
- The single-driver design is structural: `DBDriver flags.Database` is a scalar, `DockerMap` is keyed by it, `server.go` has one `db database.Service` field. Multi-database was declined three times (#235, #292, #312). [program.go L25-41](https://github.com/Melkeydev/go-blueprint/blob/81f56f8c24637d2fd2adb5f37071c6d4cb72b571/cmd/program/program.go#L25-L41), [#235](https://github.com/Melkeydev/go-blueprint/issues/235)
- History: Templates started as inline Go strings (#49), and DB and non-DB route files were separate copies until #181 merged them with `{{if}}`, which is where the 7-way duplication came from. [#49](https://github.com/Melkeydev/go-blueprint/issues/49), [#181](https://github.com/Melkeydev/go-blueprint/issues/181)

What the mainstream Go code generators do with the same mechanism: sqlc, oapi-codegen, ent, gqlgen, go-swagger, goa and mockery all render Go through `text/template`; none uses jennifer or dst. They differ only in import management (next section) and all end with in-process `format.Source` or `imports.Process`. [sqlc gen.go](https://github.com/sqlc-dev/sqlc/blob/main/internal/codegen/golang/gen.go), [oapi-codegen codegen.go](https://github.com/oapi-codegen/oapi-codegen/blob/main/pkg/codegen/codegen.go), [ent graph.go](https://github.com/ent/ent/blob/master/entc/gen/graph.go), [gqlgen templates.go](https://github.com/99designs/gqlgen/blob/master/codegen/templates/templates.go), [goa file.go](https://github.com/goadesign/goa/blob/v3/codegen/file.go)

Assessment:

- In-process formatting: `format.Source` per Go file, trivially.
- Embedded assets: `//go:embed all:templates` into one `embed.FS`, `ParseFS` with path-qualified names.
- Import management: the Template owns its import block, with each conditional import inside the same `{{if}}` as its use. Works, as go-blueprint shows, but every new Option that needs a package adds a conditional to every file that uses it.
- Maintenance burden: linear in files x Options for conditionals, and the moment a feature needs to add a line to a file "owned" by another axis the author reaches for string splicing. go-blueprint is the documented end state.
- The Template stays a near-verbatim copy of the output, which is the mechanism's real strength: reviewers can read a Template as the Skeleton file.

### B. `text/template`, fragment composition: blocks, overlays, slot contributions

How multi-axis variation is expressed: a base Template per output file declares named slots; each Choice ships a fragment set that fills slots. Two sub-variants:

1. Overlay sets via `Clone` + `Parse`. Base Templates use `{{block "db-imports" .}}{{end}}`; the Choice `postgres` ships a file defining `db-imports`, `db-open`, `db-health`. The engine clones the base set, then parses the chosen overlay sets. Last definition wins, so it composes. Precedent: oapi-codegen builds per-router output exactly this way and documents it in a source comment: "text/template permits redefining a template that has not yet been executed, so parsing the {{define}} overrides replaces the skeleton {{block}} defaults in the clone (last definition wins)". [oapi-codegen codegen.go](https://github.com/oapi-codegen/oapi-codegen/blob/main/pkg/codegen/codegen.go)
   Limit: a slot has one definition. Router, database and sessions all need to contribute to the same `imports` slot and the same `main()` body, so a one-definition slot forces slot names to be per-axis (`router-imports`, `db-imports`, `sessions-imports`), and the base Template must know every axis in advance. Adding a fourth Option means adding a slot to every base Template it touches.
2. Slot contributions. The base Template ranges over a list the engine assembled: `{{range .Slot "imports"}}{{template . $}}{{end}}`. Each Choice registers named fragments against slot names (`imports`, `main.setup`, `routes`, `makefile.targets`). Order comes from the Option catalogue, not from the Template. This is kubebuilder's marker mechanism done at render time instead of as a post-write edit (see G). Adding a fourth Option adds a fragment set and touches no existing Template unless it needs a slot that does not exist yet.

Assessment:

- In-process formatting: same as A; the rendered file is whole, `format.Source` applies. Fragment indentation does not matter for Go; for Makefile and YAML the fragment must carry its own indentation, which is the known weak spot (go-blueprint's fragments show the drift).
- Embedded assets: one `embed.FS`; a directory per Choice (`templates/choices/postgres/`) holding fragment files. `ParseFS` base-name collisions make a manual walk mandatory.
- Import management: each fragment declares its imports in an `imports` slot contribution. Duplicates (two Options importing `net/http`) must be deduplicated by the engine before rendering, or pruned afterwards with the AST pass in the next section. gqlgen and goa both chose "declare explicitly, prune unused in-process". [gqlgen prune.go](https://github.com/99designs/gqlgen/blob/master/internal/imports/prune.go), [goa file.go `finalizeGoSource`](https://github.com/goadesign/goa/blob/v3/codegen/file.go)
- Maintenance burden: fragments and base Templates agree on an implicit contract (the fragment for `main.setup` may use `mux` because the base declared it). That contract must be written down per slot; go-blueprint's unwritten version of it is where its fragment bugs (#265 fiber assets) came from. [PR #265](https://github.com/Melkeydev/go-blueprint/pull/265)
- Readability: a base Template with slots is still near-verbatim output; a fragment set for one Choice is a readable list of "what this Choice adds". Combination behaviour is invisible until rendered, which is why a golden-file test per supported Configuration matters more here than in A.
- The empty-definition rule means a Choice that wants to remove a base default must override with `{{""}}`, not an empty body. Document it or the first "turn this off" overlay will silently do nothing.

### C. templ

The owner asked whether templ could be a simpler fit. It cannot be used for Go, SQL, YAML or Makefile output.

- Whitespace is collapsed by design. The parser reduces every whitespace run after a text node to one of none, single space or newline, and the generator then demotes newline to a space: "Normalize whitespace for minified output. In HTML, a single space is equivalent to any number of spaces, tabs, or newlines." Whitespace-only text is dropped. [textparser.go](https://github.com/a-h/templ/blob/main/parser/v2/textparser.go), [types.go `TrailingSpace`](https://github.com/a-h/templ/blob/main/parser/v2/types.go), [generator.go](https://github.com/a-h/templ/blob/main/generator/generator.go)
- Verified locally with v0.3.1070: a component containing a two-target Makefile with tab-indented recipes rendered as the single line `run: go run ./cmd/web docker: docker compose up`.
- `<pre>` is not exempt (#90, #937). The whitespace-modes proposal (#1205) was closed as not planned on 2026-07-09; the maintainer: "it's not possible to implement new whitespace handling modes without significant changes to both the parser and formatter". [#937](https://github.com/a-h/templ/issues/937), [#1205](https://github.com/a-h/templ/issues/1205)
- Non-HTML output is explicitly out of scope: "there's no plan to support the generation of plain text (or YAML, or anything else) with templ templates" (#385); "Supporting formats other than HTML isn't a goal for templ" (#90). [#385](https://github.com/a-h/templ/issues/385), [#90](https://github.com/a-h/templ/issues/90)
- The grammar is hostile to code: `{` opens an expression (no text fallback, so `func main() {` is a parse error, verified locally), `<` opens an element, line-leading `if`/`for`/`switch` open statements, ` @` and `\t@` open component calls (every Makefile `\t@echo`). [stringexpressionparser.go](https://github.com/a-h/templ/blob/main/parser/v2/stringexpressionparser.go), [statements](https://templ.guide/syntax-and-usage/statements), [#631](https://github.com/a-h/templ/issues/631)
- `{ expr }` is always `html.EscapeString`-escaped (`< > & ' "`), so Go `a < b`, `&&` and SQL quotes are corrupted unless passed through `@templ.Raw`. [runtime.go](https://github.com/a-h/templ/blob/main/runtime.go), [rendering raw HTML](https://templ.guide/syntax-and-usage/rendering-raw-html)
- The only working form, verified locally, is `@templ.Raw("package main\n\nfunc main() {\n")` per line: the Template becomes Go string literals, which is strictly worse than `text/template`. templ adds no output formatting API; `go/format` would still be needed. [pkg index](https://pkg.go.dev/github.com/a-h/templ)
- Maintenance signals, for completeness: 10.5k stars, pre-1.0, 6-10 releases a year, CLI and runtime versions must be bumped together, every CLI bump churns every `_templ.go`. [v0.3.819 notes](https://github.com/a-h/templ/releases/tag/v0.3.819), [modcheck.go](https://github.com/a-h/templ/blob/main/cmd/templ/generatecmd/modcheck/modcheck.go)

templ stays relevant as a Choice inside the Skeleton (HTML templating), not as the Generator's Template language.

### D. AST assembly: `go/ast` or `dave/dst` splicing into a reference Skeleton

How multi-axis variation is expressed: the repository holds one complete, compiling reference Skeleton (the default Configuration). Each Choice is Go code that parses the relevant reference files, inserts or removes declarations and statements, adds or removes imports, and adds or deletes whole files. Non-Go files still need Template or copy handling.

- With `go/ast` directly, comment placement breaks on insertion (issue #20744 above). Acceptable only if reference files carry no comments near edit points, which conflicts with a Skeleton meant to be read by humans.
- `dst` keeps comments attached to nodes (`Decs.Start`, `Decs.Before`). Verified locally: inserting a decorated `store := sessions.NewCookieStore()` after `mux := http.NewServeMux()` kept `// Build the router.` and `// Start listening.` on their statements, with no `go` binary. [dst README](https://github.com/dave/dst)
- Import management without shell-outs: the `guess` and `simple` restorer resolvers and the `goast` decorator resolver are pure maps or import-block parsing; `gopackages` runs `go list` via `go/packages`, and `gobuild` can too in module mode. The engine would use `NewRestorerWithImports("main", guess.WithMap(knownNames))`. [resolvers](https://github.com/dave/dst#resolvers), [gopackages resolver](https://github.com/dave/dst/blob/master/decorator/resolver/gopackages/resolver.go), [go/build importGo](https://github.com/golang/go/blob/master/src/go/build/build.go)
- Fidelity bugs that would show up in a Skeleton: blank lines inside functions sometimes dropped (#65, partly fixed in v0.28.0), import grouping not preserved (#61), stdlib alias handling (#82), indentation after a comment on a multi-result return (#89). [#65](https://github.com/dave/dst/issues/65), [#61](https://github.com/dave/dst/issues/61), [#89](https://github.com/dave/dst/issues/89)
- Maintenance: 0.x, a release gap from December 2022 to April 2026, then v0.27.4 and v0.28.0 in 2026; v0.28.0 requires a Go 1.26 toolchain and `x/tools` v0.50.0. 268 known importers including DataDog orchestrion, Azure service-operator and coder/coder. [versions](https://pkg.go.dev/github.com/dave/dst?tab=versions), [go.mod](https://github.com/dave/dst/blob/master/go.mod), [importers](https://pkg.go.dev/github.com/dave/dst/decorator?tab=importedby)
- gonew is the minimal form of copy-and-rewrite: `go mod download` the source module, copy every file, rewrite the module path in `.go` files by parsing with `parser.ImportsOnly` and editing byte ranges, and `modfile` for `go.mod`. No Options, no variation, requires the `go` binary; "intentionally minimal", "an experiment". Its `fixGo` is a clean reference for AST-safe import-path rewriting. [gonew main.go](https://github.com/golang/tools/blob/master/cmd/gonew/main.go), [blog](https://go.dev/blog/gonew)

Assessment:

- In-process formatting: dst prints through `go/printer`; run `format.Source` after.
- Embedded assets: the reference Skeleton is embedded as files (`all:` needed) and parsed at generation time.
- Import management: explicit, per patch, through the resolver map; unused imports cannot arise if a patch removes the import with the usage.
- Maintenance burden: the reference Skeleton is a real Go module that `go build` and `go test` can verify directly, which no Template approach offers. Against that, every Choice is procedural code locating anchors by structure (the second declaration, the function named `main`) or by marker comment, and a reader cannot see what a Configuration produces without running the engine.
- Verbosity: the local 12-line splice needed roughly 15 lines of dst construction code for one statement plus one comment.

### E. `dave/jennifer`

How multi-axis variation is expressed: plain Go, `if cfg.Router == Chi { g.Id("r").Op(":=").Qual("github.com/go-chi/chi/v5", "NewRouter").Call() }`.

- Imports are added only for `Qual` calls that rendered, so unused imports are impossible; `Render` runs `format.Source` in-process (verified isolated, no `go` binary). Package names for non-stdlib paths are guessed from the last path element and aliased: `github.com/go-chi/chi/v5` rendered as `v5 "github.com/go-chi/chi/v5"` locally until `ImportName` was supplied. Import blocks are a single sorted group, not goimports' stdlib/third-party grouping. [jen.go](https://github.com/dave/jennifer/blob/master/jen/jen.go), [README imports](https://github.com/dave/jennifer#imports)
- Go-only. SQL, YAML, Makefile, Dockerfile need a second mechanism. A `Raw` insertion primitive has been an open request since 2018 (#50). [#50](https://github.com/dave/jennifer/issues/50)
- The Template no longer resembles the output; character count is roughly 1.5-3x the generated Go, and every token is a method call. Reviewing a Template against the intended Skeleton file requires golden tests.
- Maintenance: 3.6k stars, 1,297 known importers (dagger, syft, Encore), v1.7.1 September 2024, commits in 2026. None of sqlc, oapi-codegen, ent, gqlgen, goa, go-swagger, mockery or wire uses it. [versions](https://pkg.go.dev/github.com/dave/jennifer?tab=versions), [importers](https://pkg.go.dev/github.com/dave/jennifer/jen?tab=importedby)

Assessment: strongest guarantees on import correctness and formatting, weakest on readability and coverage of non-Go files. Not recommended for the prototype; the guarantees it gives are obtainable in B with an import-prune pass.

### F. gonew-style copy-and-rewrite

Covered under D. As a standalone engine it has no Option axis and depends on `go mod download`; kratos (`git clone` a layout repo) and create-go-app (one repository per backend Choice, `git clone`) are the same shape. Not viable alone; the "reference Skeleton" idea survives inside D. [kratos repo.go](https://github.com/go-kratos/kratos/blob/main/cmd/kratos/internal/base/repo.go), [create-go-app create.go](https://github.com/create-go-app/cli/blob/master/cmd/create.go)

### G. Composition patterns from other generators

Surveyed for how they express multi-axis output and resolve "several Options touch one file".

| Tool | Multi-axis expression | Same file touched by many | Hooks or shell-outs | Source |
|---|---|---|---|---|
| cookiecutter | Jinja `{% if %}` in contents and paths; no native conditional files (issue #723 open since 2015) | whole-file conditionals only | conditional structure requires a `post_gen_project` subprocess that deletes files | [hooks](https://cookiecutter.readthedocs.io/en/stable/advanced/hooks.html), [#723](https://github.com/cookiecutter/cookiecutter/issues/723) |
| copier | `copier.yml` questions with `type`, `choices`, `when`; conditional paths `{% if use_docker %}Dockerfile{% endif %}.jinja` (empty name is skipped); several independent templates applied to one project with separate answers files | whole-file; cross-template is overwrite | `_tasks`/`_migrations` are optional subprocesses; core rendering is in-process | [configuring](https://copier.readthedocs.io/en/stable/configuring/), [updating](https://copier.readthedocs.io/en/stable/updating/) |
| Yeoman | `composeWith` runs several generators in lock-step priorities, all writing one in-memory FS (mem-fs) | later generator `read`/`append`/`extendJSON` on the shared FS before commit; conflict resolution on disk collision | npm install optional | [file system](https://yeoman.io/authoring/file-system.html), [composability](https://yeoman.io/authoring/composability.html) |
| Nx devkit | generators are functions over a virtual `Tree`; `generateFiles` (EJS, `__name__`/`__tmpl__` filename substitution); call other generators with the same tree | `updateJson`, `addDependenciesToPackageJson`, `tree.read`/`write` on earlier output; dry-run falls out of the virtual tree | install callback optional | [Tree](https://nx.dev/reference/core-api/devkit/documents/Tree), [generateFiles](https://nx.dev/reference/core-api/devkit/documents/generateFiles) |
| Rails | `hook_for :orm, :test_framework` dispatches an axis to `<choice>:<generator>`; `--skip-*`; `OPTION_IMPLICATIONS` for `--minimal` | string insertion on disk: `insert_into_file`, `route`, `environment`, `gem`; post-hoc `remove_file` | yes in practice | [Base.hook_for](https://api.rubyonrails.org/classes/Rails/Generators/Base.html), [Thor actions](https://www.rubydoc.info/gems/thor/Thor/Actions) |
| kubebuilder machinery | `Template` and `Inserter` builders; plugins chained in a fixed order; in-memory file map, then write | `Inserter` adds deduplicated fragments immediately before `// +kubebuilder:scaffold:<marker>` lines in files earlier builders wrote | none in the library; `imports.Process` with `FormatOnly: true` on Go output | [machinery](https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/machinery), [scaffold.go](https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/machinery/scaffold.go), [markers](https://book.kubebuilder.io/reference/markers/scaffold) |
| genny / buffalo | Options add sub-generators to a `Group`; virtual `Disk`, `DryRunner`/`WetRunner`; `embed.FS` + `TemplateTransformer` | later generators find and modify files on the virtual Disk | `gogen.Fmt` calls `imports.Process` with nil options, so it shells out; buffalo cli archived 2024 | [genny](https://github.com/gobuffalo/genny), [goimports.go](https://github.com/gobuffalo/genny/blob/main/gogen/goimports/goimports.go) |
| gowebly | per-framework directory for 2 files, `if`/`switch` for everything else | whole-file | `go mod tidy`, `go fmt`, `templ generate`, npm | [create.go](https://github.com/gowebly/gowebly/blob/main/internal/actions/create.go) |
| overlay directories (edx layered cookiecutter, combino) | base directory plus per-feature layers, last wins | edx: add-only, "partial overwrites of files is not allowed"; combino: per-glob merge strategies (`replace`, `deep`, `append`) | edx layering runs in a post-gen script | [edx ADR 0003](https://github.com/openedx/edx-cookiecutters/blob/master/docs/decisions/0003-layered-cookiecutter.rst), [combino](https://cdn.jsdelivr.net/npm/combino@1.10.4/README.md) |

No Go web-application generator with per-Option overlay directories plus file merging was found. Pure overlay copying only works when layers are add-only (edx forbids partial overwrites) or when per-file merge strategies exist (combino's `append`/`deep` for JSON). For Go source, "append" is not a merge strategy; a Go file touched by three Options needs either whole-file conditionals (A), slot composition (B), or marker insertion (kubebuilder, which is B applied after the file exists).

Two transferable ideas outside the Template question:

- A virtual file tree (`map[path][]byte` or `fs.FS`) that every Choice writes into and the front end commits at the end. Yeoman, Nx, genny and kubebuilder all have one; it gives dry-run, the zip for the web UI, and byte-identical CLI/web output from one code path.
- copier's answers file: the Skeleton contains `.copier-answers.yml` generated from the questions, and `copier update` re-renders and three-way merges. The map already decided that the Skeleton embeds a copy of its Configuration; copier is the precedent that this enables upgrades later.

## Cross-cutting: import management without goimports

Every mature Go generator converged on the same rule: the Template or builder declares its imports explicitly, and the generator removes unused ones in-process; nobody lets goimports add imports. Reasons on record:

- ent must defer `imports.Process` until every file is written because "'goimports' will drop undefined package", which is exactly the state of a Skeleton mid-generation. [ent graph.go](https://github.com/ent/ent/blob/master/entc/gen/graph.go)
- gqlgen's `Prune` is "a wrapper around x/tools/imports that only removes imports, never adds new ones", implemented with `astutil.DeleteNamedImport` plus `format.Source`. [prune.go](https://github.com/99designs/gqlgen/blob/master/internal/imports/prune.go)
- goa's `finalizeGoSource` parses, detects used imports, removes the rest, `ast.SortImports`, `format.Node`, then `imports.Process` with `FormatOnly: true`. [file.go](https://github.com/goadesign/goa/blob/v3/codegen/file.go)
- Go issue #14500 (2016) added `FormatOnly` because a generator author had "seen goimports insert the wrong package" and considered a missing import "a generator bug" that should not be masked. [golang/go#14500](https://github.com/golang/go/issues/14500)
- incident.io measured templating under a second and goimports' import resolution at 15 seconds, and moved to pre-populated imports plus prune. [incident.io](https://incident.io/blog/codegen-faster)

For this Generator the prune pass (parse, `astutil.UsesImport` or a selector scan, `astutil.DeleteImport`, `format.Source`) is about 40 lines of standard library and `x/tools/go/ast/astutil`, runs with no `go` binary, and makes candidates A and B as safe on imports as E. Dedup of identical import paths across fragments belongs in the same pass.

## Recommendation for the engine composition prototype (#7)

Compare these three; the first is the control.

1. Whole-file conditional Templates (A). `text/template`, one Template per Skeleton file, Configuration as dot, imports declared in-Template, `format.Source` plus the prune pass. It is what every Go generator in the survey does and what go-blueprint shows breaking down once Options cross files. The prototype should measure exactly that: how many existing Templates the fourth Option touches.

2. Slot contributions (B, variant 2). Base Templates with named slots; each Choice ships a fragment set keyed by slot; the engine orders contributions from the Option catalogue and renders the whole file, then formats and prunes. Hypothesis to test: the fourth Option touches zero existing Templates, and the slot contract per file is short enough to document. The `Clone` + overlay variant (B.1) is worth a half-day sub-experiment because it needs no engine code, but its one-definition-per-slot limit is expected to fail the router x database x sessions case.

3. Reference Skeleton plus dst patches (D). A compiling default Skeleton in the repository, verified by `go build` and `go test` in CI, and a dst-based patch per Choice using the `guess` resolver (no `go` binary). Hypothesis to test: readability of a multi-axis file is best here because the reference file is a real file, and worst for understanding a non-default Configuration, because the patches are code. Watch dst's blank-line and import-grouping fidelity bugs and its Go 1.26 toolchain floor. Non-Go files fall back to A inside this candidate, so the prototype also shows how well two mechanisms coexist.

Excluded: templ (collapses whitespace, HTML-only by stated policy), jennifer (Go-only, Template unreadable, no mainstream precedent), gonew-style copy (no Option axis, needs `go`), `imports.Process` in add mode (spawns `go env`, host-dependent output), overlay directories as the primary mechanism (no merge strategy for Go files).

Regardless of candidate: embed with `all:`, walk the FS and name Templates by path, render into a virtual file tree committed once by the front end, and keep Option relations (requires, conflicts with, defaults to) in the declaration rather than in Templates; go-blueprint's four copies of "tailwind implies htmx" are the cautionary result of not doing so.

## Open points for later tickets

- Makefile and YAML fragments in candidate B carry semantic indentation that no formatter will fix; the prototype should include one Makefile target contributed by two Options.
- `go.sum` without a toolchain (listed in the map's "Not yet specified") is untouched by this research; none of the surveyed generators produce it without `go mod tidy`.
- `format.Source` output can change between Go versions; pinning the Generator's Go version pins the Skeleton's formatting, and a release checklist should note it.
