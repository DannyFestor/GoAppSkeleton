# Go code-quality tooling for Skeletons

Resolves [#5](https://github.com/DannyFestor/GoAppSkeleton/issues/5). Researched 2026-10-08. Every golangci-lint claim marked "verified" was run locally against golangci-lint v2.14.0 on Go 1.27.1; the three preset configs below pass `golangci-lint config verify`, `golangci-lint linters` and `golangci-lint run` on a throwaway module. GitHub counts come from the GitHub API on the same day.

## Question

Which specific tools and configurations should the spec name for the code-quality Options, so the catalogue lists tools rather than categories? What should a minimal, standard and strict golangci-lint preset enable, and how do presets stay valid as linters are added and deprecated? Which architecture-testing and mutation-testing tool should each Option name, and how is each wired into a Skeleton's Makefile and CI?

## Recommendation

| Category | Tool to name | Option and Choices | Tier (per map) |
|---|---|---|---|
| Linting | golangci-lint v2 (pinned, currently v2.14.0) | `lint-preset`: `none`, `minimal`, `standard`, `strict`; defaults to `standard` | core |
| Architecture tests | go-arch-lint v1.19.0, with depguard rules inside the strict lint preset as the zero-cost complement | `arch-test`: `none`, `go-arch-lint`; defaults to `none` | next or later |
| Mutation tests | gremlins v0.6.0, wired so the binary is one Makefile variable; re-evaluate gomutants before the Option leaves "later" | `mutation-test`: `none`, `gremlins`; defaults to `none`; requires the tests Option | later |

The three lint presets are built on three different mechanisms on purpose. `minimal` is golangci-lint's own `default: standard` group and names no linter, so it cannot break. `standard` is `default: standard` plus an explicit `enable` list of 47 linters that catch bugs or enforce idioms with low false-positive rates. `strict` is `default: all` minus a reasoned `disable` list, so every new linter arrives with the next version bump. What keeps all three valid is not the YAML but the Generator: it pins one golangci-lint version, renders every preset in its CI, and runs the real binary against a reference Skeleton. A Renovate bump that removes a linter fails the Generator's build, never the user's.

The rest of this document gives the reasoning, the concrete configs, and the wiring.

## golangci-lint

### Version facts (primary sources)

- Latest release v2.14.0, 2026-09-24. The v2 line started 2025-03-24; 2026 has had a minor release roughly every month. No v3 is announced; the roadmap has none and the open milestones are `v2.14`, `unreleased` and `zfuture`. [1][2][3]
- `version: "2"` is the only accepted config schema value. [4]
- Versioning policy: a major is reserved for "backward incompatible change of configuration with huge impact"; "any minor update may report more errors than the previous release"; and "no major version will be created when a linter is removed". So linter churn is by design a minor-version event. [3]
- Deprecation lifecycle, three phases, nominally one minor each: warning (linter still runs), then a do-nothing placeholder that `default: all` no longer enables and that "should be removed from the `disable` option", then removal. In practice it is slower: `wsl` has sat at the warning phase since v2.2.0 (2025-06-28) and still does in v2.14.0. [3][5]
- Currently deprecated, each with a replacement of the same name plus a suffix: `wsl` to `wsl_v5` (v2.2.0), `gomodguard` to `gomodguard_v2` (v2.12.0), `exhaustruct` to `exhaustruct_v5` (v2.13.0). Removed in v2.0.0: deadcode, execinquery, exhaustivestruct, exportloopref, golint, ifshort, interfacer, maligned, nosnakecase, scopelint, structcheck, tenv, varcheck; `gosimple` and `stylecheck` were merged into `staticcheck`. [2][5][6]
- `linters.default` accepts `standard`, `all`, `none`, `fast`. Verified: `standard` is exactly errcheck, govet, ineffassign, staticcheck, unused (5 linters); `fast` enables 48, including the deprecated `gomodguard` and `wsl`; `all` enables 115, including the three deprecated ones. `fast` means "needs no type information"; of the standard set only ineffassign qualifies, so `fast` is a pre-commit convenience, not a preset. [4][5][7]
- Verified behaviour on unknown and deprecated names, which drives the preset design:
  - An unknown or removed name in `enable` **or** `disable` is a hard error (`unknown linters: ...`, exit 3), for `run` and for `linters` alike.
  - A warning-phase deprecated name in `enable` runs and logs a suggested replacement config. The same name in `disable` is silent.
  - A formatter name under `linters.enable` is an error (`X is a formatter`). [8]
  - `golangci-lint config verify` validates against the JSON schema only. It rejects an unknown key (`additional properties 'bogus-key' not allowed`) but accepts an unknown linter name. Validating a preset therefore needs `golangci-lint linters` or `run`, not just `config verify`.
- Formatters are their own section: gci, gofmt, gofumpt, goimports, golines, swaggo. `run` reports unformatted files; `fmt` rewrites them; `fmt --diff` exits non-zero when anything would change. [9]
- There is no first-party recommended or strict profile. The project's own `.golangci.yml` says "This configuration file is not a recommendation." It uses `default: none` plus 33 named linters, all four exclusion presets, gofmt and goimports. The reference file lists every option for documentation only. [10][11]
- Exclusion presets: `comments` (staticcheck ST1000/ST1020-22 plus revive exported and package-comment messages), `std-error-handling` (unchecked `Close`, `Flush`, `os.Remove`, print and env calls), `common-false-positives` (gosec G103, G204, G304), `legacy`. v2 applies none by default. [6][12]
- Installation: binary install is recommended, with the pinned install script `curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.14.0`. `go install` and the `tool` directive are explicitly not recommended (result depends on local Go, can alter project deps). Homebrew works but "can use an unexpected version of Go". For CI: "it's important to have reproducible CI: don't start to fail all builds at the same time", so pin. [13][14]
- Official Action `golangci/golangci-lint-action@v9` (v9.3.0, 2026-06-29). `version:` accepts `v2.14` or `v2.14.0` or `latest`; `version-file:` reads `.golangci-lint-version` or `.tool-versions`; `verify: true` (default) runs `config verify`; `only-new-issues: true` runs `--new-from-patch` on PRs and needs `pull-requests: read`. It caches `~/.cache/golangci-lint` keyed on `go.mod`. The README recommends a lint job separate from `go test`. [15][16]

### Design principles for the presets

1. One axis, three stops. The presets differ in how much they demand, not in taste. Each stop is a superset of the one below, so a user can move up without un-learning anything.
2. A preset is a Template, not a file. The Generator renders `.golangci.yml` from the Choice plus the module path, the templating Choice, the test-library Choice and any build tags. A user who edits the rendered file owns it from then on.
3. Every Skeleton must pass its own preset on first `make lint`, including `strict`. That is a constraint on every Template in the Generator and the Generator's CI enforces it (see "Keeping presets valid").
4. No preset enables a linter whose target library no Option can pull in. Linters for testify, slog, SQL and HTTP stay, because they are no-ops until the API is used and cost nothing (testifylint "does nothing unless testify is adopted").
5. Settings only where the default is wrong for a web app. Fewer settings means fewer keys that can be renamed in a future version.

### Preset `minimal`

golangci-lint's own default group, plus gofmt and goimports so `make fmt` exists. Names no linter, so no version bump can make it reference something that no longer exists. Suits a user who wants `go vet` and staticcheck as a CI gate and nothing else.

```yaml
version: "2"
run:
  relative-path-mode: gomod
  modules-download-mode: readonly
linters:
  default: standard
  exclusions:
    generated: strict
formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - {{ .ModulePath }}
```

Enabled (verified, 5): errcheck, govet, ineffassign, staticcheck, unused. Most of this already shows in an editor through gopls (see "Overlap with gopls and go vet"), so the value here is a CI gate, not new findings.

### Preset `standard`

`default: standard` plus an explicit `enable` list. Everything added either finds a real bug, enforces an idiom most Go code already follows, or keeps tests honest. No complexity ceilings, no purity rules, no naming rules. Verified: 52 linters enabled, zero issues on a clean module.

```yaml
version: "2"
run:
  relative-path-mode: gomod
  modules-download-mode: readonly
linters:
  default: standard
  enable:
    # Bugs and resource leaks
    - asasalint
    - bidichk
    - bodyclose
    - copyloopvar
    - durationcheck
    - errchkjson
    - errorlint
    - exhaustive
    - fatcontext
    - gocheckcompilerdirectives
    - gosec
    - makezero
    - musttag
    - nilerr
    - nilnesserr
    - nilnil
    - noctx
    - reassign
    - rowserrcheck
    - sqlclosecheck
    - wastedassign
    # Idioms and modern Go
    - canonicalheader
    - dupword
    - errname
    - exptostd
    - gocritic
    - gomoddirectives
    - intrange
    - iotamixing
    - mirror
    - misspell
    - modernize
    - nakedret
    - nolintlint
    - perfsprint
    - predeclared
    - revive
    - sloglint
    - unconvert
    - unparam
    - usestdlibvars
    - whitespace
    # Tests
    - testableexamples
    - testifylint
    - thelper
    - tparallel
    - usetesting
  settings:
    errcheck:
      check-type-assertions: true
    govet:
      enable:
        - nilness
    nolintlint:
      require-explanation: true
      require-specific: true
  exclusions:
    generated: strict
    warn-unused: true
    presets:
      - comments
formatters:
  enable:
    - gofumpt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - {{ .ModulePath }}
  exclusions:
    generated: strict
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

Why these and not others:

| Group | Linters | Reason |
|---|---|---|
| Web-app bugs | bodyclose, noctx, canonicalheader, rowserrcheck, sqlclosecheck, musttag, errchkjson, gosec | The Skeleton is `net/http` plus a database Option. Unclosed bodies and rows, requests without a context, and JSON on untagged structs are the bugs a web app ships. gosec is on because a web app is exposed; the Skeleton's Templates carry a reasoned `//nolint:gosec` where G304 fires on static file serving, rather than the `common-false-positives` preset which would also silence G204. |
| Error handling | errorlint, errname, nilerr, nilnesserr, nilnil, errcheck `check-type-assertions` | Correct `errors.Is`/`As`, no `return nil, nil`, no silent type assertions. `err113` and `wrapcheck` are strict-only: both push a house style (sentinel errors, wrap at every boundary) rather than catch a bug. |
| Modern Go | modernize, intrange, copyloopvar, exptostd, perfsprint, usestdlibvars, mirror | All auto-fixable. A Skeleton targets the latest stable Go, so there is no reason to emit pre-1.22 forms. The reference config notes modernize's `forvar` overlaps copyloopvar and `rangeint` overlaps intrange; both are kept because the overlap costs nothing and modernize can be disabled per analyzer. [11] |
| gocritic, revive at defaults | gocritic default tags (diagnostic, style, performance); revive default rules | Each is one linter with a curated default set; enabling them at defaults is the lowest-noise way to get a hundred small checks. Tuning lives in strict. |
| Tests | thelper, tparallel, usetesting, testifylint, testableexamples | Keep generated and user-written tests honest. `paralleltest` is strict-only: forcing `t.Parallel()` on every test is a policy, not a bug. |
| sloglint | defaults | Skeletons are stdlib-first, so `log/slog` is the logger. Defaults only; `no-global` and `static-msg` move to strict. |
| nolintlint | require explanation and specific linter | Every suppression names what it suppresses and why. This is what makes the other 51 linters survivable. |
| Formatters | gofumpt, goimports | gofumpt is a strict superset of gofmt; `local-prefixes` groups the Skeleton's own imports. gci is skipped because it fights goimports over grouping. |
| Exclusions | `comments` preset | Without it, revive's `exported` rule demands a doc comment on every exported handler and constructor. A Skeleton's generated code is not a library. |

Deliberately left out of standard, with the reason: contextcheck and containedctx (medium false positives around `http.Request`), gochecknoglobals (a web app has a few legitimate package-level vars for templates and regexes; strict handles them), mnd, goconst, dupl, funlen, gocognit, cyclop, nestif (all ceilings; strict), depguard (needs rules; strict), ireturn, iface, interfacebloat (interface policy; strict), godot, varnamelen, tagliatelle, forbidigo, wrapcheck, err113, exhaustruct_v5, paralleltest, testpackage, gochecknoinits, godox (all policy; strict).

### Preset `strict`

`default: all` minus a reasoned `disable` list, plus thresholds. This is where TuiSnip's config was the inspiration; the differences are that web-specific names (`w`, `r`) are exempt from varnamelen, the depguard rules are module-agnostic until an Option adds layer rules, and `golines` wraps at 120 so `lll` can stay off. Verified: 93 linters enabled, deprecated linters absent from the enabled set, no deprecation warnings.

```yaml
version: "2"
run:
  relative-path-mode: gomod
  modules-download-mode: readonly
linters:
  default: all
  disable:
    # Deprecated in v2.x; the *_v2 / *_v5 successor is enabled by `all`.
    - exhaustruct
    - gomodguard
    - wsl
    # Libraries no Option pulls in.
    - arangolint
    - clickhouselint
    - ginkgolinter
    - promlinter
    - protogetter
    - spancheck
    - zerologlint
    # Duplicates of an enabled linter.
    - gocyclo        # cyclop
    - maintidx       # cyclop + gocognit + funlen
    - gomodguard_v2  # depguard
    - godoclint      # godot + revive
    # Whitespace and layout opinions with no defect signal.
    - decorder
    - goheader
    - gosmopolitan
    - lll            # golines wraps instead
    - nlreturn
    - noinlineerr
    - tagalign
    - wsl_v5
  settings:
    cyclop:
      max-complexity: 10
    depguard:
      rules:
        repo-wide:
          list-mode: lax
          files:
            - $all
          deny:
            - pkg: github.com/pkg/errors
              desc: use the standard errors package
            - pkg: io/ioutil
              desc: deprecated; use io and os
            - pkg: math/rand$
              desc: use math/rand/v2 or crypto/rand
    dupl:
      threshold: 100
    errcheck:
      check-type-assertions: true
      check-blank: true
    exhaustive:
      check:
        - switch
        - map
    exhaustruct_v5:
      enforce-patterns:
        - ^{{ .ModulePathRegex }}/.+
      allow-empty-returns: true
    funlen:
      lines: 50
      statements: 30
    gocognit:
      min-complexity: 15
    gocritic:
      enabled-tags:
        - diagnostic
        - style
        - performance
        - experimental
        - opinionated
      disabled-checks:
        - whyNoLint   # nolintlint covers it
    govet:
      enable-all: true
      disable:
        - fieldalignment
    iface:
      enable:
        - identical
        - unused
        - opaque
        - unexported
        - unusedmethod
    interfacebloat:
      max: 5
    ireturn:
      allow:
        - anon
        - error
        - empty
        - stdlib
        - generic
    nestif:
      min-complexity: 4
    nolintlint:
      require-explanation: true
      require-specific: true
    revive:
      enable-default-rules: true
      rules:
        - name: argument-limit
          arguments: [4]
        - name: function-result-limit
          arguments: [3]
        - name: flag-parameter
        - name: early-return
        - name: deep-exit
        - name: import-shadowing
        - name: unused-receiver
        - name: bare-return
        - name: use-any
    sloglint:
      no-global: all
      context: scope
      static-msg: true
    staticcheck:
      checks:
        - all
        - -ST1000
        - -ST1020
        - -ST1021
        - -ST1022
    testifylint:
      enable-all: true
    unparam:
      check-exported: true
    varnamelen:
      ignore-type-assert-ok: true
      ignore-map-index-ok: true
      ignore-chan-recv-ok: true
      ignore-names:
        - id
        - db
        - tx
        - w
        - r
  exclusions:
    generated: strict
    warn-unused: true
    presets:
      - comments
    rules:
      - path: _test\.go
        linters:
          - dupl
          - err113
          - exhaustruct_v5
          - funlen
          - goconst
          - varnamelen
          - wrapcheck
formatters:
  enable:
    - gofumpt
    - goimports
    - golines
  settings:
    gofumpt:
      module-path: {{ .ModulePath }}
    goimports:
      local-prefixes:
        - {{ .ModulePath }}
    golines:
      max-len: 120
  exclusions:
    generated: strict
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

What strict adds over standard, and why each is a policy worth a preset of its own:

| Addition | Effect on a Skeleton |
|---|---|
| cyclop 10, gocognit 15, funlen 50/30, nestif 4, revive argument-limit 4 and function-result-limit 3 | Handlers stay small; a handler that parses, validates, calls a service and renders gets split. Tests are exempt from funlen because table tests are long by nature. |
| exhaustruct_v5 scoped to the module | Every struct literal of a Skeleton type names every field. Scoped to the module so `http.Server{}` and `slog.HandlerOptions{}` are untouched; `allow-empty-returns` keeps `return User{}, err` legal. |
| iface (all analyzers), ireturn, interfacebloat 5 | Interfaces are declared at the consumer, kept to five methods, and functions return concrete types. `ireturn` allows stdlib so `http.Handler` constructors pass. |
| gochecknoglobals, gochecknoinits, forbidigo (default bans `fmt.Print*`, `print`, `println`), err113, wrapcheck, mnd, goconst, dupl | Purity rules. forbidigo's default list means a Skeleton's `main` logs through slog, which the standard preset only encourages. The Skeleton's own Templates must already obey these, otherwise strict cannot be offered. |
| paralleltest, testpackage, godot, godox, varnamelen, tagliatelle, inamedparam, funcorder, recvcheck, nonamedreturns | Policy on tests, comments, names and layout. Each is auto-fixable or a one-line fix. |
| depguard `repo-wide` | Bans `github.com/pkg/errors`, `io/ioutil` and `math/rand`. It is the hook the `arch-test` Option extends with layer rules (see "Architecture testing"). Note: depguard with no rules allows only `$gostd` everywhere, so a rule block is mandatory whenever the linter is on. [17] |
| golines 120 | Long lines get wrapped instead of reported. |

Opinions I did not adopt from TuiSnip: `importas`, `recvcheck` exclusions and the Bubble Tea `ireturn` regex are TUI-specific; `sloglint attr-only` and `key-naming-case` are a house style the Skeleton should not impose; `gocritic hugeParam` stays on because web handlers rarely take large values by copy.

### Keeping presets valid as linters come and go

Any name in `enable` or `disable` is a hard error the day that linter is removed. The golangci-lint policy makes that a minor-version event with a three-minor warning window. So the question is who notices the warning, and the answer must be the Generator, never the user.

1. **One pin.** The Generator holds a single golangci-lint version constant and renders it into the Skeleton as `.golangci-lint-version`. The Action reads that file through `version-file:`; the Makefile's `tools` target reads it for the install script. The Skeleton's embedded Configuration records which preset and which version it was rendered with. Renovate or Dependabot bumps the constant in the Generator; the Skeleton user bumps their file when they want to.
2. **Render and run in Generator CI.** For each preset and each representative Configuration (html/template and templ, each database Choice, testify on and off), the Generator's test suite renders a Skeleton, then runs the pinned binary three ways: `golangci-lint config verify` (schema, catches renamed settings keys), `golangci-lint linters` (exit 3 on an unknown linter name, verified), and `golangci-lint run` (zero issues, and stderr must not contain `is deprecated`). This is the step that turns a Renovate bump into a failing PR when a linter is removed or deprecated.
3. **Authoring rules.** `minimal` names no linter. `standard` names linters only under `enable` and `settings`. `strict` names linters only under `disable` and `settings`. Nothing ever appears in `enable` while deprecated. When a `_vN` successor appears, `strict` only needs the old name added to `disable` (it already picks the successor up through `all`), and `standard` swaps the name. When a linter is removed, the fix is deleting one line.
4. **`strict` floats, `standard` does not.** A version bump under `strict` can add a linter the Skeleton's Templates fail. That is intended; it is the Generator's test suite that fails, and the Generator either fixes the Templates or adds the linter to `disable` with a reason, before releasing. `standard` only gains linters when someone adds a line.
5. **Settings are the second risk.** A renamed settings key (`goconst.ignore-strings` became `ignore-string-values` in v2.1.0; `gofumpt.extra-rules` became `extra`) is caught by `config verify` in step 2. Keeping settings few, as the presets do, keeps this surface small. [2]
6. **No v1 compatibility.** The presets are v2-only; `golangci-lint migrate` exists for users who bring a v1 file, but the Generator never emits one. [6]

### What varies per Option in the Template

| Input | Where it lands |
|---|---|
| Module path | `goimports.local-prefixes`, `gofumpt.module-path`, `exhaustruct_v5.enforce-patterns`, depguard `allow` entries once layer rules exist |
| templ Choice | Nothing extra: templ writes `// Code generated by templ - DO NOT EDIT.`, which the `generated: strict` matcher (`^// Code generated .* DO NOT EDIT\.$`) already excludes. [12] A `paths` exclusion for `_templ\.go$` is a cheap belt-and-braces line if a future templ release changes the header. |
| sqlc or other generators | Same mechanism; no path exclusions needed while headers follow go.dev/s/generatedcode. |
| Build tags for integration tests | `run.build-tags`, otherwise tagged test files are never linted. |
| testify Choice | None; testifylint is a no-op without testify. |
| arch-test = go-arch-lint | Adds the depguard layer rules (strict only; standard has no depguard). |

### Wiring into a Skeleton

Makefile targets, with the version in one place:

```make
GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)

.PHONY: tools
tools: ## Install the pinned quality tools
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $$(go env GOPATH)/bin v$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format with the enabled formatters
	golangci-lint fmt

.PHONY: fmt-check
fmt-check: ## Fail if formatting would change anything
	golangci-lint fmt --diff
```

CI, one job separate from tests as the Action README recommends:

```yaml
lint:
  runs-on: ubuntu-latest
  permissions:
    contents: read
  steps:
    - uses: actions/checkout@v6
    - uses: actions/setup-go@v6
      with:
        go-version-file: go.mod
    - uses: golangci/golangci-lint-action@v9
      with:
        version-file: .golangci-lint-version
```

`run` already reports unformatted files, so no separate `fmt --diff` step is needed in CI; keep `make fmt-check` for local use and hooks. A pre-commit hook (lefthook or whatever the hooks ticket picks) runs `golangci-lint fmt {staged_files}`; a pre-push hook runs `golangci-lint run` on the whole module, because linters type-check packages as a unit and `--new-from-rev` misses issues in old lines that a change affects. `--fast-only` is the right flag for an editor save hook, not for CI. [8]

The install path matters more than it looks. A golangci-lint binary is compiled with one Go version, and it refuses to load a module whose `go` directive is newer than that Go. The curl script downloads a release binary built with a current Go, which is why the docs prefer it over `go install`. If a later ticket adopts mise for Skeletons, mise's `go:` backend builds with the pinned Go and sidesteps the problem the same way TuiSnip does. [13]

### Overlap with gopls and go vet

- `govet` "is roughly the same as go vet and uses its passes"; the default analyzer list matches `go vet` minus `hostport`, `stdversion` and `waitgroup`. gopls runs the same vet suite plus extras (nilness, unusedparams, unusedwrite, stdversion, and the modernize family). [18][19]
- gopls enables "slightly more than half" of staticcheck by default (most SA, S and QF checks; nearly all ST off), which is close to golangci-lint's staticcheck default of `all` minus ST1000/1003/1016/1020-22. [20]
- Consequence for the Skeleton's AGENTS.md and docs: the `minimal` preset mostly re-reports what an editor already shows, so its job is the CI gate. `standard` and `strict` add findings an editor does not show. modernize is in both the editor and the linter; that duplication is harmless because both auto-fix to the same result.

## Architecture testing

### Candidates (as of 2026-10-08)

| Tool | Model | Latest release | Last commit | Open issues | CLI or library | Verdict |
|---|---|---|---|---|---|---|
| go-arch-lint (fe3dback) | Named components and vendors, allow-list graph, optional deep scan of constructor injection, YAML v3 | v1.19.0, 2026-09-07 | 2026-09-07 | 17 | CLI; exit 0/1, `--json`, Docker image, pre-commit hook, no GitHub Action | Adopt |
| arch-go (arch-go org, formerly fdaines) | Package glob patterns with `shouldOnlyDependsOn`/`shouldNotDependsOn` split into internal, external, standard; plus function, content and naming rules; compliance threshold | v2.1.2, 2026-02-03 | 2026-02-13 (renovate since) | 11, 14 PRs mostly renovate | Both; `arch-go` CLI and `api.CheckArchitecture` in `go test` | Reject for now |
| go-cleanarch | Four fixed layer names from directory names, flags only | v1.2.1, 2021-02-18 | 2021-11-08 | 4 | CLI | Reject |
| depguard inside golangci-lint | Per-file-glob allow and deny of import prefixes | tag v2.2.1, 2025-03-06; last commit 2026-09-22 | | 8 | Linter | Complement, not a replacement |
| kcmvp/archunit, LukasNiessen/ArchUnitGo | Fluent `go test` libraries | v0.2.0-alpha1 (2025-09); v0.0.2 (2026-09-30, two months old, Go 1.26+) | | | Library | Watch |
| gau, bvwells/go-clean-arch, gocal and a dozen 2026 one-day repos | | | | | | Dead or experiments |

Sources: [21][22][23][24][25].

### Why go-arch-lint

The Generator knows a Skeleton's layout and knows exactly which third-party modules each Option pulled in. That is the shape of a `.go-arch-lint.yml`: `components` come from the layout, `vendors` come from the Options, `deps` come from the layer rules the Skeleton kind defines. No other tool has a named-component model; arch-go matches dot-separated package globs against rules, so each layer edge has to be re-spelled in every rule, and it has an open nil-pointer panic when a rule's `package:` contains `**` (#269). go-cleanarch has had no commit since 2021 and hard-codes layer directory names. The ArchUnit ports run inside `go test`, which is attractive, but one is alpha with regex bugs and the other is two months old. [21][22][23]

go-arch-lint's limits, each with a mitigation the Template applies:

- It always allows the standard library, so "the domain layer may not import `database/sql`" needs depguard. The strict lint preset's `repo-wide` depguard rule is the hook; the `arch-test` Option adds a `core-purity` rule with `list-mode: strict`, `files: ["**/internal/domain/**"]` and `allow: [$gostd, <module>/internal/domain]`. [17][24]
- `deepScan` defaults to true in v3 and treats constructor injection as a dependency, which is exactly the edge a ports-and-adapters layout inverts. TuiSnip's research confirmed `app.NewX(sqlite.NewStore())` in `main` is reported as `sqlite -> app`. Render `allow.deepScan: false` globally and enable it per driving adapter only. [24]
- A component glob with no matching directory is a hard error unless `ignoreNotFoundComponents: true`. The Generator renders only components whose directories it emitted, so the flag can stay false.
- Each component must list itself in `mayDependOn`, or its own subpackages cannot import each other. The Template does this for every component.
- `_test.go` files are excluded by convention (`excludeFiles: ["^.*_test\\.go$"]`), so tests stay unconstrained.
- Needs Go 1.25+ to build, no Homebrew formula, no first-party Action; Windows path handling has an open issue (#79). A `v4` branch exists but stalled in October 2024; all 2026 releases are v1.x on master. [21]

### Wiring

```make
GO_ARCH_LINT_VERSION := 1.19.0

tools: ...
	go install github.com/fe3dback/go-arch-lint@v$(GO_ARCH_LINT_VERSION)

.PHONY: arch-lint
arch-lint: ## Check the layer rules
	go-arch-lint check
```

`go install pkg@version` run from inside the Skeleton does not touch its `go.mod`, which keeps the "a dependency arrives only because an Option pulled it in" rule intact. The CI job is a sibling of `lint`: checkout, setup-go, `make tools`, `make arch-lint`; exit code 1 on any warning, so no flags are needed. The Docker image `fe3dback/go-arch-lint` is an alternative when the runner's Go is older than 1.25, but a Skeleton on the latest stable Go never is. `go-arch-lint graph` renders an SVG of the component graph, which the Generator can mention in the Skeleton's docs but should not gate on. [21]

What the Generator renders for a ports-and-adapters Skeleton (layout per the engine ticket; the names here are placeholders):

```yaml
version: 3
workdir: .
allow:
  depOnAnyVendor: false
  deepScan: false
  ignoreNotFoundComponents: false
excludeFiles:
  - "^.*_test\\.go$"
vendors:
  {{ range .Vendors }}{{ .Name }}: { in: [ {{ .ImportGlobs }} ] }
  {{ end }}
components:
  domain:   { in: internal/domain/** }
  app:      { in: internal/app/** }
  http:     { in: internal/adapters/http/** }
  storage:  { in: internal/adapters/{{ .StorageDir }}/** }
  main:     { in: cmd/{{ .BinaryName }} }
commonComponents: [ domain ]
deps:
  domain:  { mayDependOn: [ domain ] }
  app:     { mayDependOn: [ app ] }
  http:    { mayDependOn: [ http, app ], canUse: [ {{ .HTTPVendors }} ], deepScan: true }
  storage: { mayDependOn: [ storage ], canUse: [ {{ .StorageVendors }} ] }
  main:    { anyProjectDeps: true, anyVendorDeps: true }
```

Each Option that adds a module contributes a vendor entry and the component that may use it. That is the one place where this Option interacts with every other one, and it is why `arch-test` belongs in "next" rather than core: it needs the layout and the vendor list to be stable first.

## Mutation testing

### Candidates (as of 2026-10-08)

| Tool | Mechanism | Latest release | Last commit | Open issues | Quality gate | Verdict |
|---|---|---|---|---|---|---|
| gremlins | AST mutation in a temp copy of the module, coverage-guided, one `go test` per covered mutant | v0.6.0, 2025-12-06 | 2026-03-30 | 28, plus 32 unmerged PRs | exit 10 (efficacy) and 11 (coverage); CLI flag path broken (#216), `<=` off-by-one (#248, #280) | Adopt, report-only, with workarounds |
| ooze | Library run via `go test -tags=mutation`, temp repo copy per mutant, full suite per mutant | tag v0.3.1, 2023-05-04, no GitHub releases | 2026-08-19 | 5 | `WithMinimumThreshold`, default 1.0 | Fallback |
| gomutants | `go test -overlay` patches, coverage and per-test routing, content-addressed cache, gremlins-compatible JSON and exit codes, composite Action | v0.6.1, 2026-09-10 | 2026-10-06 | 5 | exit 10/11 | Watch; swap-in candidate |
| gomu (sivchari) | Overlay-based, incremental via git, `--ci-mode --threshold` | v0.2.1, 2026-05-29 | 2026-10-05 | 11 | `--fail-on-gate` | Watch |
| mutago (quality-gates) | go-mutesting lineage, exec per mutant, optional coverage | v2.10.23, 2026-10-07, near-daily releases | 2026-10-07 | 23 | `--min-msi`, exit 4 | Too churny to pin |
| avito-tech/go-mutesting | Exec script per mutant, no coverage, no parallelism | v2.3.1, 2025-12-26 | 2026-10-02 | 11 | none (#16 open since 2022) | Reject; two open `go/types` panics on Go 1.21+ and 1.26 unanswered |
| zimmski/go-mutesting | Original; `go 1.10`, pre-generics x/tools | v1.2, 2021-06-10 | 2021-06-10 | 34 | none | Reject; "is go-mutesting dead?" open since 2022 |

Sources: [26][27][28][29][30][31][32].

### Why gremlins, and why report-only

gremlins is the only candidate with a release history longer than a year, a Homebrew tap, a Docker image, an official Action, a JSON report format that other tools now copy, and a user base that has shaken out the sharp edges. It is also stalled: one maintainer, nothing merged since March 2026, and the two bugs that matter for a CI gate are both open with fixes waiting in PRs. TuiSnip runs it in anger (its committed report: 4 mutants, 3 killed, 75% efficacy, 4 seconds) and documented the operational traps. [26][33]

The traps, and what the Template does about each:

| Trap | Source | Mitigation in the rendered Skeleton |
|---|---|---|
| Threshold flags on the CLI never produce a non-zero exit (#216); only `.gremlins.yaml` thresholds work. And `--threshold-efficacy N` fails at exactly N because of `<=` (#248, #280). | [34][35] | Ship a `.gremlins.yaml` with thresholds at 0 (report only, the map's own ruling) and document how to raise them there. Never gate a PR on it. |
| Integration mode `-i` reports killed mutants as lived because it never passes `-count=1` (#272, fix in unmerged PR #293). | [36] | Do not render `-i`. Per-package mode is the default and is correct. |
| A second SIGTERM during shutdown panics with `send on closed channel` (PR #283, unmerged). A CI job timeout sends SIGTERM then SIGKILL. | [37] | Set the job `timeout-minutes` well above the expected run and cap memory instead of relying on the job timeout. |
| A mutant that reverses a loop counter can allocate without bound; TuiSnip saw GitHub kill the runner. | [33] | Run under `systemd-run --scope -p MemoryMax=12G -p MemorySwapMax=0 -p OOMPolicy=continue` on the Ubuntu runner, as TuiSnip does. The kernel kills the one test process and gremlins counts the mutant as killed. |
| Generated files are mutated unless excluded; only `_test.go` is excluded by default. | [26] | Render `exclude-files` from the Options that generate code (templ, sqlc, mocks). |
| Type constraints with `\|` are mis-mutated as INVERT_BITWISE (#252). Third-party benchmark claims v0.6.0 panics under Go 1.26.x; no matching issue in the tracker. | [38][39] | Verify with one run on the Generator's reference Skeleton at the pinned Go version before the Option ships; this is the "re-evaluate gomutants" trigger. |
| Runs "can take hours" on big modules; a fresh Skeleton takes seconds. | [26] | `--diff origin/$GITHUB_BASE_REF` with `fetch-depth: 0` is the documented way to keep PR runs short, but `--diff` has three open issues about silently skipping mutants (#278, #296, #301), so it stays a local convenience, not a CI mode. |

gomutants is the one to watch. It patches through `go test -overlay` so the source tree is never copied or modified, routes mutants to the tests that cover them, caches results between runs, and emits gremlins-compatible JSON with the same exit codes. If it is still alive in six months, swapping the binary is a one-variable change in the Makefile below, which is why the wiring should not mention gremlins anywhere except that variable. It needs Go 1.26+, which a Skeleton on latest stable already has. ooze is the fallback if a build-tag gate inside `go test` is ever wanted, at the price of a `go.mod` dependency and a three-year-old tag. [28][29]

### Wiring

```make
MUTATION_TOOL        := gremlins
GREMLINS_VERSION     := 0.6.0
MUTATION_REPORT      ?= mutation-report.json
MUTATION_EXCLUDE     := {{ .GeneratedFileRegex }}
# Each mutant recompiles under parallel workers; the baseline run does not account for that.
MUTATION_TIMEOUT_COEFFICIENT := 5
PKG                  ?= ./...

tools: ...
	go install github.com/go-gremlins/gremlins/cmd/gremlins@v$(GREMLINS_VERSION)

.PHONY: test-mutation
test-mutation: ## Write the mutation report (PKG=./internal/... for one package)
	$(MUTATION_TOOL) unleash --exclude-files '$(MUTATION_EXCLUDE)' \
		--timeout-coefficient $(MUTATION_TIMEOUT_COEFFICIENT) --output $(MUTATION_REPORT) $(PKG)
```

`.gremlins.yaml` rendered alongside, thresholds at zero and the five default operators on:

```yaml
unleash:
  threshold:
    efficacy: 0
    mutant-coverage: 0
```

CI is its own workflow, `workflow_dispatch` plus an optional weekly cron, never a required check, with the report as an artifact:

```yaml
name: Mutation
on:
  workflow_dispatch:
  schedule:
    - cron: "0 3 * * 1"
permissions:
  contents: read
jobs:
  mutation:
    runs-on: ubuntu-latest
    timeout-minutes: 120
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
      - run: make tools
      - run: |
          sudo --preserve-env env "PATH=$PATH" systemd-run --scope --quiet --uid="$(id -u)" --gid="$(id -g)" \
            -p MemoryMax=12G -p MemorySwapMax=0 -p OOMPolicy=continue make test-mutation
      - if: always()
        uses: actions/upload-artifact@v7
        with:
          name: mutation-report
          path: mutation-report.json
```

The official `go-gremlins/gremlins-action@v1` is not used: it runs on the deprecated Node 16 runtime (v1.0.3, May 2024), only wraps `gremlins unleash`, and has an open issue about unauthenticated binary downloads hitting rate limits. `go install` with a pinned version does the same job with no moving parts. [40]

## Catalogue entries

Draft declarations in the option model's three relations, for the option-model ticket to refine:

| Option | Choices | defaults to | requires | conflicts with |
|---|---|---|---|---|
| `lint-preset` | `none`, `minimal`, `standard`, `strict` | `standard` | | |
| `arch-test` | `none`, `go-arch-lint` | `none` | a layered layout (whatever the engine ticket names it) | |
| `mutation-test` | `none`, `gremlins` | `none` | `tests` not `none` | |

`arch-test = go-arch-lint` with `lint-preset = minimal` or `standard` renders go-arch-lint without the depguard complement, so stdlib bans in the domain layer are not enforced. That is acceptable (the layer edges are still checked) but the Skeleton's docs should say so; alternatively the option model could make `arch-test` default `lint-preset` to `strict`. Flagged under open questions.

The map lists mutation testing and architecture tests in "next or later". This research does not change that: `arch-test` needs the layout and vendor list to be stable, and `mutation-test` needs the tests Option and a verified run on the current Go.

## Open questions for the human

1. Should `lint-preset` have a `none` Choice at all, given the map lists golangci-lint in core? `none` costs nothing and keeps the Option honest for users who bring their own config.
2. `standard` keeps gosec on. For a web Skeleton that serves static files, G304 will fire inside the Generator's own Templates and needs a `//nolint:gosec // reason` there. Accept that, or move gosec to strict?
3. `strict` excludes `comments` (no doc comment required on exported identifiers), same as TuiSnip. A public generator's users may expect godoc enforcement in strict. Keep the exclusion, or drop it and enable `godoclint`?
4. Should `arch-test = go-arch-lint` force `lint-preset = strict` (so depguard's domain-purity rule exists), or document the gap?
5. Thresholds in strict copy TuiSnip's (cyclop 10, gocognit 15, funlen 50/30, nestif 4, interfacebloat 5, argument-limit 4). They were tuned for a TUI. Keep as the starting point, or loosen funlen for handler-heavy code?
6. The mutation job's memory cap uses `systemd-run`, which exists only on the Ubuntu runner. Is Linux-only CI for the mutation job acceptable for Skeletons?
7. Re-evaluation date for gomutants versus gremlins: propose the day the `mutation-test` Option leaves "later", with one run of each on the reference Skeleton as the test.

## Sources

1. golangci-lint releases: https://github.com/golangci/golangci-lint/releases
2. golangci-lint changelog: https://golangci-lint.run/docs/product/changelog/
3. golangci-lint roadmap (versioning, deprecation phases): https://golangci-lint.run/docs/product/roadmap/
4. Configuration file reference: https://golangci-lint.run/docs/configuration/file/
5. Linter registry source (groups, deprecations, fast flags): https://github.com/golangci/golangci-lint/blob/main/pkg/lint/lintersdb/builder_linter.go
6. Migration guide v1 to v2: https://golangci-lint.run/docs/product/migration-guide/
7. `linters.default` enum in the JSON schema: https://golangci-lint.run/jsonschema/golangci.jsonschema.json
8. CLI reference: https://golangci-lint.run/docs/configuration/cli/
9. Formatters: https://golangci-lint.run/docs/formatters/
10. golangci-lint's own config: https://github.com/golangci/golangci-lint/blob/main/.golangci.yml
11. Reference config: https://github.com/golangci/golangci-lint/blob/main/.golangci.reference.yml
12. Exclusion presets source: https://github.com/golangci/golangci-lint/blob/main/pkg/result/processors/exclusion_presets.go
13. Local install: https://golangci-lint.run/docs/welcome/install/local/
14. CI install: https://golangci-lint.run/docs/welcome/install/ci/
15. golangci-lint-action README: https://github.com/golangci/golangci-lint-action
16. golangci-lint-action inputs: https://github.com/golangci/golangci-lint-action/blob/main/action.yml
17. depguard README: https://github.com/OpenPeeDeeP/depguard
18. Linters list and govet description: https://golangci-lint.run/docs/linters/
19. gopls analyzers: https://github.com/golang/tools/blob/master/gopls/doc/analyzers.md
20. gopls settings (staticcheck): https://github.com/golang/tools/blob/master/gopls/doc/settings.md
21. go-arch-lint repo, syntax and releases: https://github.com/fe3dback/go-arch-lint and https://github.com/fe3dback/go-arch-lint/blob/master/docs/syntax/README.md
22. arch-go repo and config struct: https://github.com/arch-go/arch-go and https://github.com/arch-go/arch-go/blob/main/api/configuration/config.go
23. go-cleanarch: https://github.com/roblaszczak/go-cleanarch
24. TuiSnip go-arch-lint research (local verification of deep scan and self-dependency behaviour, 2026-09-30): `../TuiSnip/docs/research/go-arch-lint.md`
25. kcmvp/archunit and ArchUnitGo: https://github.com/kcmvp/archunit and https://github.com/LukasNiessen/ArchUnitGo
26. gremlins README and docs: https://github.com/go-gremlins/gremlins and https://gremlins.dev/latest/usage/commands/unleash/
27. gremlins releases: https://github.com/go-gremlins/gremlins/releases
28. gomutants: https://github.com/gomutants/gomutants
29. ooze: https://github.com/gtramontina/ooze
30. gomu: https://github.com/sivchari/gomu
31. mutago: https://github.com/quality-gates/mutago
32. go-mutesting original and avito fork: https://github.com/zimmski/go-mutesting (issue #100) and https://github.com/avito-tech/go-mutesting (issues #16, #37, #44)
33. TuiSnip toolchain notes and committed `mutation-report.json`: `../TuiSnip/docs/toolchain.md`, `../TuiSnip/mutation-report.json`
34. gremlins #216 threshold flags never fail: https://github.com/go-gremlins/gremlins/issues/216
35. gremlins PRs #248 and #280 (`<=` threshold comparison): https://github.com/go-gremlins/gremlins/pull/280
36. gremlins #272 and PR #293 (integration mode, `-count=1`): https://github.com/go-gremlins/gremlins/issues/272
37. gremlins PR #283 (signal handling): https://github.com/go-gremlins/gremlins/pull/283
38. gremlins #252 (generics): https://github.com/go-gremlins/gremlins/issues/252
39. gomutants performance notes claiming a gremlins panic under Go 1.26: https://github.com/gomutants/gomutants/blob/main/docs/performance.md
40. gremlins-action: https://github.com/go-gremlins/gremlins-action
