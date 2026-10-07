# Pairwise Configuration testing: tooling and CI cost

Research for [#4](https://github.com/DannyFestor/GoAppSkeleton/issues/4), part of the wayfinder map [#1](https://github.com/DannyFestor/GoAppSkeleton/issues/1). Date: 2026-10-08.

The question: how does the Generator prove that every Configuration it accepts yields a Skeleton that builds, vets and passes its own tests, when the full matrix is far too large to enumerate?

Model assumed throughout, the core tier as sketched in the ticket: app type (2), database (4), router (5), config source (2), log format (3), session store (4), and twelve yes/no Options. That is 18 Options and 2 x 4 x 5 x 2 x 3 x 4 x 2^12 = 3,932,160 Configurations before constraints.

## Recommendation in one paragraph

Commit roughly 25 pairwise Configuration files plus 4 to 6 named Configurations with golden files. Run all of them from one Go test in the Generator's own module, each Configuration as a parallel subtest that generates into a temp dir and runs `go build`, `go vet` and `go test -count=1`. One CI job, around three minutes with a warm cache, split into shards only once it passes five. Generate the pairwise rows at development time with a small greedy generator inside the repo (about 200 lines, constraint predicate reused from the Option validator), or with PICT if you would rather not own that code; either way the test checks coverage of the committed rows itself, so the choice of generator is low-stakes. A failing Configuration is reproduced locally by its file name. Three-way coverage (80 to 100 rows) runs nightly, not per PR.

## 1. Why pairwise is the right strength

NIST SP 800-142 states the empirical basis: "Most failures are induced by single factor faults or by the joint combinatorial effect (interaction) of two factors, with progressively fewer failures induced by interactions between three or more factors." In the NASA data, 93% of failures were triggered by 2-way combinations and 98% by 3-way, with the curves reaching 100% at 4 to 6 ([SP 800-142, p. 5](https://nvlpubs.nist.gov/nistpubs/legacy/sp/nistspecialpublication800-142.pdf)).

The size law matters more for us than the fault data. The number of t-way tests "is proportional to v^t log n, for n parameters with v possible values each" (same document, pp. 9-10). So adding yes/no Options barely grows the pairwise set; the two largest Option cardinalities set the floor. For our model that floor is router x database = 5 x 4 = 20 rows, because every one of those 20 pairs needs its own row. Three tools (gopair, Jenny-go, PICT) all produced 20 or 21 rows for the unconstrained model, so 20 is tight. Three-way is bounded below by 5 x 4 x 4 = 80 (router x database x session). On the model without the session Option (bound 60) PICT produced 76 and Jenny-go 60, so expect 80 to 100 rows with it.

Put differently, a Generator with a well-constrained Option catalogue proves pairwise interaction coverage of its whole core tier in about as many builds as go-blueprint spends on one router.

NIST's own covering-array tables cover fixed-level arrays only ([math.nist.gov/coveringarrays](https://math.nist.gov/coveringarrays/)); there is no published optimum for a mixed-level 5^1 4^2 3^1 2^14 array, and "except in this special case for t=2 and v=2 it is generally unknown what are the sizes of optimal covering arrays" ([NIST](https://math.nist.gov/coveringarrays/coveringarray.html)). The 20-row result is minimal by the product bound, not by table lookup.

## 2. Tooling that can generate the rows

### 2.1 Go libraries: the ecosystem is close to empty

Searches on pkg.go.dev for "pairwise", "covering array" and "combinatorial testing" return distance metrics, syzkaller coverage packages and nothing else ([pairwise](https://pkg.go.dev/search?q=pairwise&m=package), [covering array](https://pkg.go.dev/search?q=covering+array&m=package), [combinatorial testing](https://pkg.go.dev/search?q=combinatorial+testing&m=package)). GitHub repo, code and topic searches found four repositories.

| Repo | What it is | Constraints | License | Activity | Verdict |
|---|---|---|---|---|---|
| [Warashi/gopair](https://github.com/Warashi/gopair) | Importable library, `Seeds{"A":2,...}.Generate(order)`, IPOG plus orthogonal-array fast path plus density greedy, N-way | None (opaque value indices, no predicate) | None declared; pkg.go.dev hides docs "due to license restrictions" | Last push 2026-07-19, 0 stars, no tags | Unusable until licensed; no constraints anyway |
| [chas678/Jenny-go](https://github.com/chas678/Jenny-go) | Go port of Bob Jenkins' jenny, CLI only, `package main` | Forbidden tuples via `-w` flags (`-w1a2cd`) | MIT | Last push 2026-07-08, 0 stars | go.mod declares `module jennygo`, so `go run ...@latest` fails; must vendor source. Shell-out only |
| [cybergarage/go-pict](https://github.com/cybergarage/go-pict) | Parser for PICT's TSV output | n/a | Apache-2.0 | v1.0.2, 2025-11 | Fine as a parser; does not generate |
| [oneumyvakin/orthogo](https://github.com/oneumyvakin/orthogo) | 116 lines, 2-way, pre-expanded rows | None | Apache-2.0 | Last commit 2018 | Dead |

No Go code on GitHub shells out to `pict` (`gh search code 'exec.Command "pict"' --language go` returns nothing). Whatever we do here, we are the first.

### 2.2 Microsoft PICT

[microsoft/pict](https://github.com/microsoft/pict) is MIT, actively pushed (2026-09), 1.4k stars. The only release asset is `pict.exe`; Linux and macOS build from source with CMake or the plain Makefile. Measured: `make -j` on the v3.7.4 tarball took 5.7 s wall on Apple Silicon; `brew install pict` takes 4.5 s from the bottled [Homebrew formula](https://formulae.brew.sh/formula/pict) (bottles exist for Linux x86_64 and ARM64 too). No Debian package, no official GitHub Action, no published container image (there is a `Containerfile` in the repo but nothing is pushed anywhere). Running it on our 17-parameter model takes 6 to 12 ms.

The model file ([doc/pict.md](https://github.com/microsoft/pict/blob/main/doc/pict.md)) is three sections: parameters (`Router: stdlib, chi, echo, gin, fiber`), optional sub-models, and constraints. Options that matter to us: `/o:N` sets the strength (default 2), `/e:file` seeds the output with existing rows so new rows extend rather than replace the set, `/r:N` randomises with a printed seed, `/b:N` tries N seeds and keeps the smallest. Output is tab-separated with a header line, deterministic without `/r`.

Calling PICT from CI is cheap, but it is a second toolchain. Either `brew install pict` on a macOS runner, or `git clone` + `make` on ubuntu (cmake and g++ are preinstalled), 6 to 10 s per job. My view: do not run it in CI at all. Run it at development time, commit the rows, and let CI only check the committed rows (section 4). Then PICT is a developer convenience, not a CI dependency.

### 2.3 NIST ACTS, for comparison only

Java, public domain, GUI and CLI ([NIST download page](https://csrc.nist.gov/Projects/automated-combinatorial-testing-for-software/Downloadable-Tools)). The Basic jar is a direct download; the Advanced version, which is the one with constraints, requires an email request. Not a fit for an MIT Go project's test suite.

### 2.4 Writing our own

I prototyped a greedy generator (AETG-style: build candidate rows one Option at a time, pick the Choice that covers the most uncovered pairs, keep the best of 30 candidates, repeat until all feasible pairs are covered) with a `valid(Row) bool` constraint predicate and a backtracking fill that detects infeasible pairs. It is 196 lines of plain Go, no dependencies. On the 18-Option model with three constraints (one requires, two conflicts) it produced 24 rows, best of 20 seeds, in 7.9 s total (unoptimised; the tie-break function is O(pairs) per call and could be memoised). All rows satisfied the constraints; 4 of 907 pairs were correctly reported infeasible. PICT produced 22 rows on a similar constrained model, Jenny-go 21. Two or three extra rows cost 10 s of CI; the generator costs 200 lines we own.

The argument for owning it: the constraint predicate is exactly the Option validator the map already requires ("the CLI, web form, validator and docs all derive from that declaration"). PICT would need the requires/conflicts relations translated into its `IF [X] = "on" THEN [Y] = "y1";` syntax, a second encoding of the same facts that can drift. With an in-repo generator the pairwise rows are derived from the single Option declaration, which is the whole point of that declaration.

The argument against: it is code to maintain and a greedy algorithm gives no optimality guarantee. For 18 Options that guarantee is worth two rows.

## 3. Expressing requires and conflicts

Every tool with constraints reduces them to one of two forms, and both of ours map directly.

| Relation in our Option declaration | PICT ([doc](https://github.com/microsoft/pict/blob/main/doc/pict.md)) | NIST ACTS ([user guide](https://csrc.nist.gov/csrc/media/Projects/automated-combinatorial-testing-for-software/documents/acts_user_guide_for_basic1.0_and_advanced3.3_versions_nov23.pdf)) | Jenny forbidden tuples | Predicate |
|---|---|---|---|---|
| X=a requires Y=b | `IF [X] = "a" THEN [Y] = "b";` | `X = "a" => Y = "b"` | forbid X=a with each Y != b | `!(x == a && y != b)` |
| X=a conflicts with Y=b | `IF [X] = "a" THEN [Y] <> "b";` | `X = "a" => Y != "b"` | forbid X=a with Y=b | `!(x == a && y == b)` |

"Defaults to" is not a constraint; it does not change the covering array. The generator must know which pairs are infeasible under the constraints and exclude them from the coverage target, otherwise it runs forever. PICT and ACTS do this internally; Jenny prints `Could not cover tuple`; the prototype computes feasibility by trying to complete each pair to a full valid row.

One subtlety worth a line in the option model spec: requires chains. If A requires B and B requires C, the predicate handles it for free (both rules are checked on every row). PICT also handles it, because it evaluates all constraints on every candidate. Forbidden-tuple tools need the transitive closure spelled out.

## 4. Golden files for named Configurations

Pairwise proves that Skeletons compile and pass their tests. It says nothing about whether the files are what we meant. Golden files do that, for a handful of Configurations a human has read.

### 4.1 Format: one txtar archive per named Configuration

`golang.org/x/tools/txtar` is the format the Go team built for exactly this. Its stated goals are to "be trivial enough to create and edit by hand", "be able to store trees of text files describing go command test cases", and "diff nicely in git history and code reviews" ([package doc](https://pkg.go.dev/golang.org/x/tools/txtar)). `cmd/go`'s own tests are txtar archives under `testdata/script`, each holding a whole module tree ([README](https://raw.githubusercontent.com/golang/go/master/src/cmd/go/testdata/script/README)). Non-goals that bite: no binary data, no file modes, no symlinks. A Skeleton with an executable script needs its mode recorded separately. A file containing a line that looks like `-- name --` breaks `Format`; `rogpeppe/go-internal/txtar` adds `Quote`/`Unquote` for that case.

A golden directory (`testdata/<name>/...`) is the alternative. `gotest.tools/v3/fs` can compare a directory against a manifest ([doc](https://pkg.go.dev/gotest.tools/v3/fs)), but it has no update flag for whole directories and a reviewer has to open each file. One archive per Configuration keeps the entire Skeleton in one diff hunk set. I prefer txtar.

Note that `go help packages` says "The go tool will ignore a directory named "testdata"" ([cmd/go](https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns)), so `.go` files inside golden archives or directories never confuse `go build ./...` on the Generator.

### 4.2 Update flag

The stdlib pattern is a package-level flag and nothing else:

```go
var update = flag.Bool("update", false, "update .golden files")
```

That line is verbatim from [`cmd/gofmt/gofmt_test.go`](https://raw.githubusercontent.com/golang/go/master/src/cmd/gofmt/gofmt_test.go); [`go/printer/printer_test.go`](https://raw.githubusercontent.com/golang/go/master/src/go/printer/printer_test.go) has the same. `go test ./internal/skeleton -update` rewrites, CI runs without the flag and fails on any mismatch. The libraries add little over that for a single archive compare:

- [`gotest.tools/v3/golden`](https://pkg.go.dev/gotest.tools/v3/golden), Apache-2.0, v3.5.2 (Sep 2024). `golden.Assert`, `golden.AssertBytes`, `golden.Get`. Registers `-update` only if no other package did ([source](https://raw.githubusercontent.com/gotestyourself/gotest.tools/main/internal/source/update.go)). Normalises CRLF to LF by default.
- [`sebdah/goldie/v2`](https://pkg.go.dev/github.com/sebdah/goldie/v2), MIT, v2.8.0 (Oct 2025). `goldie.New(t).Assert(t, name, bytes)`, diff engines, per-subtest fixture dirs. Registers `-update` unconditionally, so it panics if you also declare the flag yourself.

Both are single-blob oriented; neither compares directory trees. Use the stdlib flag plus txtar and skip the dependency. `github.com/google/go-cmp` is fine for a readable failure message, but its doc says "Do not depend on this output being stable" ([cmp](https://pkg.go.dev/github.com/google/go-cmp/cmp)), so never persist its output.

### 4.3 Gotchas, mostly determinism

- `-update` is passed to every package's test binary under `go test ./... -update`, so a package without the flag errors with "flag provided but not defined". Declare the flag in one shared test helper or scope the command to the package.
- Map iteration order is unspecified ([spec](https://go.dev/ref/spec)); sort every map before rendering and sort the directory walk before archiving. The engine is a pure function from Configuration to files (map Note), which also means no timestamps or host-dependent strings in Templates. Generator version strings in output should come from a fixed value in tests.
- Emit `\n` unconditionally and add `* text=auto eol=lf` to `.gitattributes` so Windows checkouts do not flip the goldens ([gotest.tools golden doc](https://pkg.go.dev/gotest.tools/v3/golden) gives the same advice).
- A missing golden file should fail with a hint to run `-update`, not silently create one.
- Golden updates belong in the same commit as the Template change that caused them; the golden diff is the review artefact. No primary source states this as a rule; it follows from the mechanics.

## 5. CI cost

### 5.1 go-blueprint, the reference for what not to do

Confirmed from the workflow files and the Actions API ([generate-linter-core.yml](https://raw.githubusercontent.com/Melkeydev/go-blueprint/main/.github/workflows/generate-linter-core.yml), [generate-linter-advanced.yml](https://raw.githubusercontent.com/Melkeydev/go-blueprint/main/.github/workflows/generate-linter-advanced.yml), [testcontainers.yml](https://raw.githubusercontent.com/Melkeydev/go-blueprint/main/.github/workflows/testcontainers.yml)):

```yaml
matrix:
  framework: [chi, gin, fiber, gorilla/mux, httprouter, standard-library, echo]
  driver: [mysql, postgres, sqlite, mongo, redis, scylla, none]
  git: [commit, stage, skip]
```

7 x 7 x 3 = 147 jobs (the API reports `total_count: 147` for run 29566613538), plus 42 "advanced" jobs (7 frameworks x 6 features, one `--feature` per job, driver pinned to postgres), plus 5 testcontainer jobs on fiber only. 194 jobs per PR.

What each job runs on the generated project: `golangci-lint run`. Nothing else. No `go build`, no `go vet`, no `go test` (lint type-checks, so a compile error surfaces, but Skeleton tests never execute). Features are never combined, so htmx plus tailwind plus docker is untested. The `git` dimension triples the matrix to test three values of an Option that cannot interact with compilation.

Per-job timing from the API: 54 to 71 s wall, of which 32 to 41 s is `go install golangci-lint@v1.63.4` from source in every job, 7 to 10 s is setup-go, and 3 to 4 s is the lint itself. The setup-go cache is keyed on the generator's `go.sum`, not the generated project's, and every log read says `Cache is not found`. `fail-fast` defaults to true ([workflow syntax](https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions)), so one failing cell cancels the rest: run 19064668695 shows 1 failure and 146 cancelled. Every retained run of both linter workflows ended in failure. The matrix has never produced a full pass/fail picture.

Lessons, in order of cost: pay runner overhead once rather than 147 times; install tools from prebuilt binaries or not at all; cache against the Skeletons' dependencies; `fail-fast: false`; test interactions, not dimensions; run the Skeleton's tests.

### 5.2 Platform facts

All from GitHub Docs.

- Standard runners for public repos are 4 vCPU, 16 GB ([runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)); private repos get 2 vCPU.
- Actions minutes are free for public repos on standard runners ([billing](https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions)).
- A matrix produces at most 256 jobs per run; Free accounts run 20 jobs concurrently ([limits](https://docs.github.com/en/actions/reference/limits)).
- Cache limit 10 GB per repo; entries unused for 7 days are evicted ([caching](https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/caching-dependencies-to-speed-up-workflows)).
- `actions/setup-go` caches both `GOMODCACHE` and `GOCACHE` ([package-managers.ts](https://raw.githubusercontent.com/actions/setup-go/main/src/package-managers.ts)). The key hashes `go.sum` in v5 and `go.mod` in v6 and later; `cache-dependency-path` overrides it ([README](https://raw.githubusercontent.com/actions/setup-go/main/README.md)). For us the key must include whatever file pins the Skeletons' dependency versions, otherwise the cache never refreshes when a pinned version changes.
- Observed cache restore on a hit: 80 MB in 3 s including the whole setup-go step (jackc/pgx [job 111202785206](https://github.com/jackc/pgx/actions/runs/37123046430/job/111202785206)).

### 5.3 Measured per-Skeleton cost

Measured on an M1 Pro with Go 1.27.1 on a one-handler `net/http` module with one `httptest` test, then with chi v5 and pgx v5 imported. Expect 4 vCPU Linux to be 1.5 to 3x slower on the CPU-bound cold compile; the warm numbers are dominated by process startup and linking and move less.

| Scenario | `go build ./...` | `go vet ./...` | `go test ./...` |
|---|---|---|---|
| stdlib only, cold GOCACHE | 4.3 s | 1.4 s | 2.2 s |
| chi + pgx, cold GOCACHE, warm module cache | 5.7 s | 1.4 s | 2.1 s |
| chi + pgx, warm GOCACHE | 0.10 s | 0.15 s | 0.16 s (cached) |
| `go mod download` of chi + pgx and transitive deps, cold | 1.35 s (55 MB) | | |

The cold cost is the standard library compiling (130 MB of GOCACHE for a stdlib-only project). Dependencies add little once the module cache is warm.

The number that decides the strategy: the build cache is keyed on content, not path. `go help cache` says the cache "correctly accounts for changes to Go source files, compilers, compiler options, and so on" ([cmd/go](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching)); it does not mention paths, so this was verified. Copying the module to a second directory with a different module path and building against the same fresh GOCACHE: the first build compiled 235 packages in 5.8 s, the second compiled only its own package in 0.5 s. Vet facts and test-only std packages (`net/http/httptest`) were a further 1.7 s and 2.0 s once, then 0.55 s for `go test -count=1`.

So N Skeletons that pin the same dependency versions, built in one job, compile std and every dependency once. Each further Skeleton costs roughly one second here, call it 3 to 4 s on a 4 vCPU runner. Use `-count=1`; test result caching does not carry across module paths and we want the tests to actually run.

### 5.4 How to batch

| Shape | Wall time, N=40 | Runner-seconds | Failure attribution |
|---|---|---|---|
| (a) one matrix job per Configuration | ~62 s (two waves of 20 on Free) | ~1,240 s | Best: one job per name |
| (b) one job, shell loop | ~183 s | ~183 s | Worst: one log, first failure aborts or you parse |
| (c) K=4 shards via matrix | ~63 s | ~252 s | A shard fails; the loop must name the Configuration |
| (d) one job, Go subtests with `t.Parallel()` | ~183 s, less with overlap | ~183 s | One named subtest per Configuration, `go test -json` output |

Assumptions: 15 s fixed overhead per job (set up, checkout, setup-go, cache restore and save, from the logs above), 12 s cold first build on 4 vCPU (zero if setup-go restored a GOCACHE), 4 s per further Skeleton. With the ticket's pessimistic 8 s per Skeleton, (b)/(d) become about 6 minutes and 4 shards about 1.8 minutes.

(d) is (a)'s attribution at (b)'s cost, and it also runs locally. `testing.T.Parallel` runs up to `-parallel` subtests at once, defaulting to `GOMAXPROCS` ([testing](https://pkg.go.dev/testing#T.Parallel), [go help testflag](https://pkg.go.dev/cmd/go#hdr-Testing_flags)). Each subtest spawns `go build`, which itself uses all cores, so the parallel gain is in overlapping link, spawn and I/O rather than compile; the build cache is "safe for concurrent invocations of the go command" ([cmd/go](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching)). Shard with (c) when the single job exceeds five minutes, using `fail-fast: false`, and keep the subtest structure inside each shard.

## 6. Recommended strategy

### 6.1 Three layers

1. **Validator unit tests.** Pure, no toolchain. Every requires/conflicts relation has a test; the pairwise rows are checked for validity against the same validator. Milliseconds.
2. **Pairwise build set, per PR.** About 25 committed Configuration files under `testdata/configurations/pairwise/`, one per row, named by index and a short slug (`row-07-api-postgres-chi.yaml`; final name format follows the Configuration file ticket). A test asserts that these rows still cover every feasible pair of the current Option catalogue; that check is 40 lines and independent of which tool produced the rows, which is what makes the generator choice low-stakes. If a new Option lands without updated rows, this test fails with the list of uncovered pairs. Each row then runs generate, `go build ./...`, `go vet ./...`, `go test -count=1 ./...` in a temp dir as a parallel subtest.
3. **Named golden Configurations.** Four to six Configurations a human has read line by line, stored as txtar under `testdata/configurations/golden/<name>.txtar`, each with its Configuration file next to it. Proposed set, to be confirmed when the option model exists: `default` (what you get by answering nothing: stdlib router, html/template, HTMX, the map's dogfooded Skeleton), `minimal` (every Option at its smallest Choice, no database, no extras), `everything` (every yes/no Option on, picking the Choices with the most Templates), `api-postgres-chi` and `web-sqlite-templ` as two realistic middles. These also run build/vet/test, so they are both golden tests and five more rows of the build set.

Pairwise covers interaction bugs between any two Options: a Template that references a database import the router Choice does not provide, a config-source Choice that drops a variable a log-format Choice expects. Golden files cover intent: the right files with the right content for Configurations people actually use. Neither covers the other's job.

### 6.2 Job count and runtime

One workflow, two jobs per PR: `generator` (lint and unit tests of the Generator itself, under a minute) and `skeletons` (layers 2 and 3 above, about 30 Skeletons). Estimated 2 to 4 minutes for `skeletons` with a warm cache, 15 to 20 s per shard-run of setup if sharded later. A scheduled nightly job runs the three-way set (80 to 100 rows, 6 to 9 minutes), which catches the few faults that need three interacting Options without putting them on the PR path.

Compare: go-blueprint spends roughly 190 jobs and 3 runner-hours per PR to lint 194 Skeletons with no feature combinations and no tests.

### 6.3 Generating and refreshing the rows

Rows are regenerated when the Option catalogue changes, not on every test run. Two ways, choose one when the option model lands:

- In-repo greedy generator behind the same `-update` flag as the goldens (`go test ./internal/matrix -update` rewrites `testdata/configurations/pairwise/`). Deterministic seed, best-of-N. Reuses the validator as the constraint predicate. ~200 lines.
- PICT at development time: a small `go generate` step renders the Option declaration into a `.pict` model (requires and conflicts become `IF/THEN` lines), runs `pict model.txt /e:existing.tsv` so existing rows are kept and only new pairs add rows, and writes the Configuration files. Requires `brew install pict` locally; CI never runs it.

Either way the coverage test in layer 2 is the gate, and the review artefact is the diff of committed Configuration files.

### 6.4 Reproducing a failure locally

The subtest is named after the Configuration file, and the failure message prints the exact commands. A CI failure in `TestSkeletons/pairwise/row-07-api-postgres-chi` is reproduced with either

```sh
go test ./internal/matrix -run 'TestSkeletons/pairwise/row-07' -v
```

or, without the test harness,

```sh
skeleton generate --config testdata/configurations/pairwise/row-07-api-postgres-chi.yaml --out /tmp/row-07
cd /tmp/row-07 && go build ./... && go vet ./... && go test -count=1 ./...
```

The second form is also what the test prints on failure, so it can be pasted. Because the Configuration file is committed and the engine is a pure function, the Skeleton is byte-identical locally and in CI. The CLI name and flag names above are placeholders until the naming item in the map is settled.

### 6.5 What this assumes about Skeleton tests

Layer 2 runs `go test` on every Skeleton without services. That means a Skeleton's default test suite must pass with no database, no Redis and no Docker; integration tests that need them sit behind a build tag or a `make itest` target, as go-blueprint's testcontainers job does for five of its drivers. This belongs to the "Skeleton test suite shape" item in the map's not-yet-specified list and should be settled with it, not here.

## 7. Open questions for later tickets

- Whether to own the generator or use PICT (section 2.4 argues for owning it; a prototype ticket could decide in an afternoon).
- The exact named golden set, once the option model fixes defaults.
- Whether `go vet` alone is enough on Skeletons or whether golangci-lint with the shipped preset should run on the golden five (prebuilt binary, not `go install`).
- Cache key design once dependency pinning (another not-yet-specified item) decides where pinned versions live.

## Side effects of this research

The tooling investigation ran `brew install pict` (3.7.4) on the development machine and left scratch directories under `/tmp/pairwise-try`, `/tmp/pairwise-research`, `/tmp/pairwise-own` and `/tmp/ci-cost-try`. Nothing in the repository was touched apart from this file.

## Sources

Combinatorial testing theory

- NIST SP 800-142, Practical Combinatorial Testing: https://nvlpubs.nist.gov/nistpubs/legacy/sp/nistspecialpublication800-142.pdf
- NIST covering array tables: https://math.nist.gov/coveringarrays/ and https://math.nist.gov/coveringarrays/coveringarray.html
- NIST ITL Bulletin May 2016: https://csrc.nist.gov/files/pubs/shared/itlb/itlbul2016-05.pdf

Generators

- gopair: https://github.com/Warashi/gopair, https://pkg.go.dev/github.com/Warashi/gopair
- Jenny-go: https://github.com/chas678/Jenny-go; original jenny: https://burtleburtle.net/bob/math/jenny.html
- go-pict parser: https://github.com/cybergarage/go-pict
- orthogo: https://github.com/oneumyvakin/orthogo
- PICT: https://github.com/microsoft/pict, model doc https://github.com/microsoft/pict/blob/main/doc/pict.md, Homebrew https://formulae.brew.sh/formula/pict
- NIST ACTS: https://csrc.nist.gov/Projects/automated-combinatorial-testing-for-software/Downloadable-Tools and the user guide linked in section 3

Golden files

- gofmt test: https://raw.githubusercontent.com/golang/go/master/src/cmd/gofmt/gofmt_test.go
- go/printer test: https://raw.githubusercontent.com/golang/go/master/src/go/printer/printer_test.go
- testdata convention: https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns
- txtar: https://pkg.go.dev/golang.org/x/tools/txtar; go-internal txtar and testscript: https://pkg.go.dev/github.com/rogpeppe/go-internal/txtar, https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript
- cmd/go script tests: https://raw.githubusercontent.com/golang/go/master/src/cmd/go/testdata/script/README
- gotest.tools golden and fs: https://pkg.go.dev/gotest.tools/v3/golden, https://pkg.go.dev/gotest.tools/v3/fs, update flag source https://raw.githubusercontent.com/gotestyourself/gotest.tools/main/internal/source/update.go
- goldie: https://pkg.go.dev/github.com/sebdah/goldie/v2, https://raw.githubusercontent.com/sebdah/goldie/master/goldie.go
- go-cmp: https://pkg.go.dev/github.com/google/go-cmp/cmp
- Go spec, map iteration: https://go.dev/ref/spec

CI

- go-blueprint workflows: https://github.com/Melkeydev/go-blueprint/tree/main/.github/workflows; runs via https://api.github.com/repos/Melkeydev/go-blueprint/actions/runs
- GitHub runners: https://docs.github.com/en/actions/reference/runners/github-hosted-runners
- GitHub limits: https://docs.github.com/en/actions/reference/limits
- Caching: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/caching-dependencies-to-speed-up-workflows
- Billing: https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions
- Workflow syntax (matrix, fail-fast): https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions
- actions/setup-go: https://raw.githubusercontent.com/actions/setup-go/main/README.md, https://raw.githubusercontent.com/actions/setup-go/main/src/package-managers.ts
- Go build cache: https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching
- testing.T.Parallel and -parallel: https://pkg.go.dev/testing#T.Parallel, https://pkg.go.dev/cmd/go#hdr-Testing_flags
- pgx job showing cache restore timing: https://github.com/jackc/pgx/actions/runs/37123046430/job/111202785206
