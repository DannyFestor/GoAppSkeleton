# Producing go.sum without a toolchain, and keeping pinned versions fresh

Research for the ticket "Producing go.sum without a toolchain, and keeping pinned dependency versions fresh". Verified 2026-10-08 against Go 1.27.1, the Go Modules Reference, the `cmd/go` source shipped with Go 1.27.1, and the Renovate and GitHub docs. Every behaviour marked "observed" was reproduced locally with the four dependency-bearing Choices from the engine composition prototype: chi `v5.3.2`, pgx `v5.11.0`, modernc sqlite `v1.60.1` and go-redis `v9.23.0`.

Vocabulary follows `GLOSSARY.md`.

## Answer

1. The engine can emit an exact `go.sum`, and a `go.mod` that `go mod tidy` leaves unchanged, without running `go`. It does this by merging per-Choice data captured at Generator build time. The capture is not "each Choice tidied on its own". Each Choice is tidied against one catalogue-wide version table.
2. The version table is a real Go module in the Generator repo. Call it the dependency module. Its `go.mod` requires every dependency-bearing Choice's packages at once. MVS over that module picks one version per module path, the highest any Choice needs. Pinning every Choice to that table removes the cases where combining Choices would change selected versions.
3. A capture tool, run by `go generate` in Generator CI, tidies each Choice against the table. It records the Choice's require lines, its `go.sum` lines, the modules that provide its packages, and the `go mod graph` edges. The engine takes the union of the selected Choices' data and drops one kind of redundant `// indirect` line (rule below). It writes the files with `golang.org/x/mod/modfile` and `module.Sort`.
4. Fragment sets stop hand-pinning versions in the `go.mod.require` slot. The slot is filled from the captured data. Versions live in one place, the dependency module's `go.mod`.
5. To refresh, an ordinary bot (Dependabot or Renovate) or a scheduled job bumps the dependency module's `go.mod`. A Generator CI job then re-runs the capture, fails if the committed capture is stale, and builds the golden and pairwise Configurations offline. A scheduled Generator workflow that does all three steps and opens a PR is the simplest option, because neither bot can run the capture on a hosted default setup (details in "Refreshing pinned versions").

Observed: all 15 non-empty subsets of the four Choices, merged this way, load offline with `GOPROXY=off GOFLAGS=-mod=readonly` and are clean under `go mod tidy -diff`. The four-Choice merge also passes `go build`, `go vet` and `go mod verify`. Without the redundant-line rule, 4 of the 15 have one surplus `// indirect` line.

## What go.sum has to contain

- Each line is `path version hash`. A version ending in `/go.mod` hashes only that module's `go.mod`. Otherwise the hash covers the files in the module zip. `h1:` is SHA-256 via [`golang.org/x/mod/sumdb/dirhash`](https://pkg.go.dev/golang.org/x/mod/sumdb/dirhash) ([ref/mod#go-sum-files](https://go.dev/ref/mod#go-sum-files), [#authenticating](https://go.dev/ref/mod#authenticating)).
- `go mod tidy` keeps a `/go.mod` hash for every module version whose requirements appear in the module graph, selected or not, "so that the build list itself is reproducible". It also keeps zip hashes for every module whose path is a prefix of a loaded package path, so that ambiguous imports can be ruled out (`keepSums` in [`cmd/go/internal/modload/init.go`](https://go.googlesource.com/go/+/refs/tags/go1.27.1/src/cmd/go/internal/modload/init.go)).
- `go mod tidy` loads "all of the packages they import, recursively. This includes packages imported by tests (including tests in other modules)", with every build tag except `ignore` enabled ([ref/mod#go-mod-tidy](https://go.dev/ref/mod#go-mod-tidy)). So `go.sum` carries hashes for the test dependencies of dependencies. Observed: the sqlite Choice's `go.sum` holds zip hashes for `golang.org/x/tools`, `golang.org/x/sync` and `modernc.org/ccgo/v4`. `go mod why` traces them to `modernc.org/libc.test`.
- Line order is `module.Sort` over (path, version), then hashes sorted as strings (`WriteGoSum` in [`cmd/go/internal/modfetch/fetch.go`](https://go.googlesource.com/go/+/refs/tags/go1.27.1/src/cmd/go/internal/modfetch/fetch.go)). `golang.org/x/mod/module.Sort` is public, so the engine can reproduce the order byte for byte. A plain `sort -u` cannot: `tidy -diff` flagged `v1.11.1` sorting after `v1.3.0`.

## What go.mod has to contain (`// indirect`)

- At `go 1.17` and above, `go.mod` has a require "for each module that provides any package imported (even indirectly) by a package or test in the main module". This enables module graph pruning ([ref/mod#go-mod-file-require](https://go.dev/ref/mod#go-mod-file-require), [#graph-pruning](https://go.dev/ref/mod#graph-pruning)).
- `tidyPrunedRoots` ([`cmd/go/internal/modload/buildlist.go`](https://go.googlesource.com/go/+/refs/tags/go1.27.1/src/cmd/go/internal/modload/buildlist.go)) adds three kinds of root:
  - one for every module providing a package in `all`;
  - one for a module providing a test dependency of a package in `all`, when the pruned graph would otherwise select a lower version than before (tidy does not downgrade);
  - one where an import would otherwise become ambiguous ([go.dev/issue/60313](https://go.dev/issue/60313)).

  The second kind is a "version holder": an `// indirect` line for a module that provides no package to the Skeleton.
- Since Go 1.16, `go build` acts as if `-mod=readonly` were set: "if any changes to `go.mod` are needed, the `go` command reports an error" ([ref/mod#go-mod-file-updates](https://go.dev/ref/mod#go-mod-file-updates)).

## What happens when go.sum or go.mod is incomplete (observed, Go 1.27.1, offline module cache)

| Shipped | `go build` | `GOFLAGS=-mod=mod go build` | `go mod tidy -diff` |
|---|---|---|---|
| No `go.sum` | `missing go.sum entry for module providing package github.com/jackc/pgx/v5/pgxpool ...; to add: go get example.com/app` | Succeeds. It fetches, checks the checksum database and writes a `go.sum` with only the build's 32 lines (tidy wants 74 for sqlite plus pgx). | Non-empty |
| `go.mod` with only the direct requires | `go: updates to go.mod needed; to update it: go mod tidy` | Succeeds and rewrites `go.mod` | Non-empty |
| Indirect require below the version MVS selects | `updates to go.mod needed` | Succeeds and rewrites | Non-empty |
| Two `require` lines for one path at different versions | `updates to go.mod needed` | Succeeds and rewrites | Wants the extra line removed |
| Extra `go.sum` lines nobody needs | Builds | Builds | Wants them removed |

So omitting `go.sum`, or shipping a partial one, makes the user run `go mod tidy` with network access before the first build. Surplus lines build but fail any `go mod tidy -diff` check. autostrada's `make audit` runs exactly that check (see `docs/research/prior-art.md`), and a Skeleton audit target would likely do the same. A missing `go.sum` with `-mod=mod` still verifies against the checksum database: "If the `go.sum` file is not present, or if it doesn't contain a hash for the downloaded file, the `go` command may verify the hash using the checksum database" ([ref/mod#authenticating](https://go.dev/ref/mod#authenticating)).

## Why a plain union of per-Choice results is not enough

MVS selects, for each module path, the highest version required anywhere in the graph ([ref/mod#minimal-version-selection](https://go.dev/ref/mod#minimal-version-selection)). Combining Choices merges their graphs, so the selected versions can rise above what any single Choice selected.

Observed with each Choice tidied on its own from its pinned direct require:

- pgx alone selects `golang.org/x/sync v0.17.0` (an `// indirect` line).
- sqlite alone does not list x/sync at all, but its graph contains `v0.23.0`.
- pgx plus sqlite selects `v0.23.0`.

Taking the higher of the listed versions gives `v0.17.0`, and `go build` then fails with "updates to go.mod needed". Union of the separate `go.sum` files:

- it has surplus lines (x/sync `v0.17.0`, testify `v1.11.1`, go-spew, yaml), so `tidy -diff` is not clean;
- here it contained every line actually needed, but only because sqlite's test-of-dependency closure happened to include x/sync `v0.23.0`. If a Choice's graph names a higher version without importing that module, its `go.sum` has the `/go.mod` hash but not the zip hash, and the union is missing a line.

So per-Choice data needs a shared version context, or the Generator needs resolution per Configuration.

## Recommended mechanism

### At Generator build time (CI, has a toolchain)

1. **Dependency module.** A nested module, for example `catalogue/deps/`, with a `go.mod` and generated files that blank-import every third-party package each dependency-bearing Choice imports, in both normal and `_test.go` files. The always-on base Skeleton counts as one more Choice. `go mod tidy` there gives the catalogue-wide version table: one version per path, the highest that any Choice needs.
2. **Capture per Choice.** For each Choice, a scratch module starts from the dependency module's `go.mod`, imports only that Choice's packages (plus the base Skeleton's), and runs `go mod tidy`. Tidy drops requirements that provide no relevant package but never downgrades what remains (`tidyPrunedRoots`, the `gover.ModCompare(...) < 0` check). Every captured version therefore matches the table. Record:
   - the require lines (direct and `// indirect`);
   - the `go.sum` lines;
   - the modules providing packages to that Choice, from `go list -deps -test -f '{{.Module.Path}}' ./...`;
   - `go mod graph` edges.

   The capture runs with the default `GOSUMDB`, so every hash recorded has been checked against `sum.golang.org` ([ref/mod#checksum-database](https://go.dev/ref/mod#checksum-database)).
3. **Embed** the capture in the Generator, for example one small file per Choice next to its fragment set.

### At generation time (engine, pure)

1. Union the selected Choices' require lines. Versions already agree, but keep the higher one by `semver.Compare` as a safety net. A path is direct if any Choice lists it as direct.
2. Drop implied version holders. Remove an `// indirect` require for a module that provides no package to any selected Choice when another kept require's `go.mod` (from the captured edges) already requires that module at the same version or higher. This mirrors the second root kind above. In the observed case, pgx captured `github.com/stretchr/testify v1.12.1 // indirect` as a version holder because the table's `go.uber.org/atomic` (from go-redis) raises testify. When go-redis is also selected, atomic is a root and tidy drops the testify line.
3. Union the `go.sum` lines, dedupe, and order with `module.Sort` then string sort per module.
4. Write `go.mod` with `modfile.File.SetRequireSeparateIndirect`, `Cleanup` and `Format`. All of this is [`golang.org/x/mod`](https://pkg.go.dev/golang.org/x/mod): `modfile`, `module`, `semver`.

The rule in step 2 checks only one level of edges. That matches pruned graphs for `go 1.17`+ dependencies, but it is not a reimplementation of tidy: dependencies below `go 1.17` (unpruned) and the ambiguity roots are not modelled. It held for all 15 subsets here. Golden and pairwise Configurations in CI guard it (next section). If a real catalogue breaks it, the fallback is to capture per dependency profile (alternatives below).

### Guard in Generator CI

For every golden or pairwise Configuration: render, then with `GOPROXY=off GOFLAGS=-mod=readonly` run `go build ./...`, `go vet ./...`, `go test ./...`, `go mod verify` and `go mod tidy -diff`. Offline mode proves the shipped `go.sum` is complete. `tidy -diff` proves the files are exactly what tidy would write. The prototype's `build_test.go` currently runs `go mod tidy` before building only to obtain `go.sum`. Under this design it must not; it runs `tidy -diff` instead.

The `go` directive in the base `go.mod` must be at least every dependency's `go` directive. The capture can assert this from the captured `go.mod` files, and `tidy -diff` would catch it anyway.

## Alternatives considered

- **Hash lines per Choice with versions pinned per Choice (no shared table).** Rejected: the x/sync case shows combined Configurations get go.mod errors and surplus or missing `go.sum` lines.
- **Precomputed result per dependency profile.** Run tidy at build time for every distinct set of dependency-bearing Choices across valid Configurations and embed each result, deduplicated through a shared line pool. Exact by construction and needs no rule. The prototype has 12 such profiles, but the count multiplies with every dependency-bearing Option. Keep it as the fallback if the rule in step 2 proves insufficient, or use it only for profiles CI finds that the rule gets wrong.
- **MVS in process over embedded `go.mod` files.** `cmd/go/internal/mvs` and `modload` are internal. Reimplementing pruning, test-dependency roots and ambiguity checks means re-deriving tidy, and package-level import data at each selected version would also be needed. Rejected as too much surface for no gain over capture.
- **Deriving hashes from the checksum database.** `sum.golang.org/lookup/M@V` returns the `go.sum` lines for one module version, both zip and `/go.mod` (observed for `golang.org/x/sync@v0.23.0`, matching the captured lines). It does not tell you which versions MVS selects, so it only replaces the hashing step. At generation time it would also break the no-network, pure engine. Not needed: the capture already gets hashes that the go command verified against the database.
- **Omit `go.sum`, or ship `go.mod` with direct requires only, and tell users to run `go mod tidy`.** This works with network access but breaks the first offline `go build` and needs a manual step. go-blueprint does it the other way round, shelling out to `go get -u` and `go mod tidy` at generation time, and its prior-art write-up records the resulting version churn.

## Refreshing pinned versions

The dependency module's `go.mod` is the only place versions live, and it is an ordinary `go.mod` at a real path. Both bots can find it.

- **Dependabot** (`package-ecosystem: gomod`, `directories` with globs, `groups` to batch updates into one PR, `allow` with `indirect` supported for gomod) ([options reference](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference)). It cannot run the capture. Workflows triggered by Dependabot PRs get a read-only `GITHUB_TOKEN` and only Dependabot secrets ([troubleshooting Dependabot on Actions](https://docs.github.com/en/code-security/dependabot/troubleshooting-dependabot/troubleshooting-dependabot-on-github-actions)). A follow-up workflow would need raised `permissions` or a separate trigger to commit the regenerated capture.
- **Renovate** (gomod manager matches `/(^|/)go\.mod$/`; `postUpdateOptions: ["gomodTidy"]`; indirect updates disabled by default) ([gomod manager](https://docs.renovatebot.com/modules/manager/gomod/)). `postUpgradeTasks` could run the capture before committing, but tasks are "blocked by default for security reasons" and each command must match the global-only `allowedCommands`. On Mend-hosted apps this depends on the plan ([configuration options](https://docs.renovatebot.com/configuration-options/#postupgradetasks), [self-hosted configuration](https://docs.renovatebot.com/self-hosted-configuration/#allowedcommands)).
- **Scheduled Generator workflow** (recommended). On a weekly cron:
  1. run `go get -u ./... && go mod tidy` in the dependency module;
  2. run the capture (`go generate`);
  3. render and build the golden and pairwise Configurations offline as above;
  4. open one PR with `gh pr create`.

  One job does what the bots split across two systems, using tools the Generator CI already has. Dependabot alerts (vulnerability notices, separate from version updates) still work on the dependency module's `go.mod`.
- **Every PR**, a CI check re-runs the capture and fails on any diff. That catches a hand-edited version without a regenerated capture, whoever made the edit.

A release checklist adds nothing the CI check does not already enforce.

## Open questions

1. Is a clean `go mod tidy -diff` a hard requirement for every Skeleton? It decides whether the redundant-line rule, or the per-profile fallback, is needed at all. Without either, Skeletons build offline but can carry a surplus `// indirect` line.
2. Pinning every Choice to the catalogue-wide maximum can give a Choice a newer transitive version than it would get alone. For example, pgx-only Skeletons get x/sync `v0.23.0` instead of `v0.17.0`. Acceptable?
3. Two Choices of the same Option are never combined, but they share the table. If one Choice breaks at a version another needs, the table cannot hold both. Leave this to CI (the pairwise build fails), or let the dependency module be split per conflict?
4. Developer tools pinned with `tool` directives (golangci-lint, Air, sqlc, migrate) bring large requirement graphs into the Skeleton's `go.mod` and `go.sum`. The same capture handles them via `tool` lines in the dependency module, but this interacts with the quality-tooling Options.
5. Dependabot, Renovate, or the scheduled workflow: an owner preference. The recommendation is the scheduled workflow.
