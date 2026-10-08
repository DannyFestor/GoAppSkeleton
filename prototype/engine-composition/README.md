# PROTOTYPE: engine composition styles (throwaway, never merged)

Answers [Engine composition prototype: three interacting Options in competing styles](https://github.com/DannyFestor/GoAppSkeleton/issues/7), on the map [GoAppSkeleton generator spec](https://github.com/DannyFestor/GoAppSkeleton/issues/1).

Question: which Template composition mechanism keeps the engine maintainable when one Skeleton file varies along several Options at once?

The prototype skill offers a logic branch (an HTML state demo) and a UI branch; neither fits an engine question. The ticket asks for a throwaway engine, so this is plain Go.

## Run it

```sh
go run ./cmd/render -style slots -set router=chi -set database=postgres -set sessions=database -set cache=redis
go run ./cmd/render -style refpatch -set database=sqlite -out /tmp/skeleton
go test ./...                 # every style byte-identical to the control, for all 32 valid Configurations
go test -tags build ./styles  # every Configuration passes go mod tidy, go build and go vet (needs network for modules)
```

## What is here

- `config/`: the Option catalogue (router, database, sessions, cache) with **requires** and **defaults to**, a validator, and an enumerator of valid Configurations. Shared by every style.
- `vfs/`: the in-memory Skeleton and the one finishing pass every style goes through: regroup imports (std, third-party, module-local), prune unused imports, gofmt, normalise trailing newlines. In-process, no `go` binary.
- `styles/wholefile/`: **A**, the control. One Template per Skeleton file, every axis an inline `{{if}}`.
- `styles/slots/`: **B**. Base Templates declare slots; each selected Choice ships a fragment set (`<option>/<choice>/fragments.tmpl` plus whole `files/`). `<option>/_any/` applies to every Choice but none. Two kinds of definition: **slots** take any number of contributions in catalogue order (`lines`, `blocks`, `words`, `wrap`); **providers** need exactly one definition among the selected Choices (`use "database.placeholder" 1`, `route "GET" "/" "home()"`).
- `styles/overlay/`: **B.1**, the half-day sub-experiment. `Clone` plus `{{block}}` overrides, `handler.go` only.
- `styles/refpatch/`: **D**. A reference default Skeleton in `testdata/reference`, one dst patch per Choice, and non-Go files falling back to style A Templates.

Commits are one per step, so `git show --stat` on the "fourth Option" commits measures how far the cache Option reached into each style.

## Results

All four styles render byte-identical output for all 32 valid Configurations, and every Configuration builds and vets.

### Adding a fourth Option (cache: none, redis)

| Style | Existing files edited | New files | Notes |
|---|---|---|---|
| A whole-file | 6 Templates (go.mod, main.go, handler.go, Makefile, compose.yaml, .env.example) | 1 | New compound conditions: `or` gates on go.mod, Makefile and compose, plus an `and` of database and cache just to place a blank line between two YAML services |
| B slots | 0 | 2 (fragments, cache.go) | Contributed to existing slots: imports, setup, dependencies, health checks, compose services and names, env, go.mod require |
| B.1 overlay | the base Template (3 new per-axis blocks: `cache-imports`, `cache-dependencies`, `cache-health`) | 1 | Applied to the whole Skeleton, every base file the Option reaches would need new blocks, as in A |
| D refpatch | 4 non-Go Templates (same edits as A), the patch order list, the dst package-name map | 2 (58-line patch, cache.go) | Insertion points had to be anchored relative to other patches' output |

### Readability of a multi-axis file (`handler.go`)

- A: 127 lines, 43 template actions. The chi × sessions route block is a nested cross-product.
- B: base 37 lines, 13 actions, near-verbatim Go. Each Choice's fragment file reads as "what this Choice adds". Combination behaviour is visible only when rendered.
- B.1: 48 lines, but the block names encode every Option axis (`sessions-routes`), and the cookie and database session overlays duplicate the same handler functions because there is no shared Option-level set.
- D: the reference is the best-reading file (33 lines of plain Go). But the per-Choice changes are about 330 lines of AST surgery, and a non-default Skeleton exists only after replaying up to five patches in order.

### Formatting in-process

- gofmt alone is not enough. It keeps blank lines at the start and end of function bodies, and a blank line inside a struct splits its field alignment, so Go slots inside functions need trim markers (`{{- with lines "x"}}`).
- Imports: the finishing pass regroups and prunes, so a fragment contributes an import line without knowing its group. The prune guesses package names from paths. `astutil.UsesImport` guesses `v5` for chi/v5 and dropped a used import, so the pass now follows goimports' assumed-name rule. dst's `guess` resolver has the same flaw and needs a per-package map.
- Makefile and YAML: fragments carry their own indentation; only newlines are trimmed, so recipe tabs survive. Not hit as a bug, but nothing checks it.

### How declared relations plug in

- All styles: the Configuration is validated against the declaration before rendering. No style re-checks relations.
- A: the relations leak back into Templates as compound conditions (`or`, `and`).
- B: fragment-set selection is the declaration (`<option>/<choice>` plus `_any`). A **provider** used across Options is an implicit **requires**. `sessions/database` calls `use "database.placeholder"`, and with no database selected rendering fails with "want exactly one provider, got 0". The engine could check provider usage against declared relations at Generator test time.
- D: patch order is hand-maintained (router/chi must run last because it rewrites routes other patches add), and the sqlite-placeholder patch is keyed on two Selections: a relation that lives outside the declaration.

### Other findings

- `//go:embed` refuses a directory holding its own `go.mod`, so D's reference is not a buildable tree in place and has no `go.mod`.
- A whole-file "empty render means omit" rule works for A and B base files.
- B has engine code A lacks (about 240 lines against 60): fragment-set loading, slot joins, providers.

## Verdict (proposed, awaiting the owner)

B, slot contributions plus providers. It is the only style where the fourth Option touched no existing Template. Its fragment layout maps one-to-one onto the Option declaration, and the shared finishing pass absorbs the whitespace and import problems. The open cost is the slot contract: each base Template's slots and each provider's meaning must be written down, since combination behaviour is invisible until rendered.
