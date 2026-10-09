# PROTOTYPE: web UI

Throwaway. Answers the [Web UI prototype](https://github.com/DannyFestor/GoAppSkeleton/issues/11) wayfinder ticket.

```sh
cd prototype/web-ui && go run . serve --open
```

There are three variants on one route, `/?variant=A|B|C`. Switch with the floating bar or the ← → keys.

| Variant | Layout | Unmet requires | Inapplicable Options |
|---|---|---|---|
| A, Long form, live sidebar | every group on one page, with a sticky sidebar holding the errors, downloads, CLI command and Configuration file | Choice disabled, reason inline | hidden, listed under the group |
| B, Stepper | module path first, one catalogue group per step, then Review with the downloads | Choice left out, "Left out: …" note | hidden, "Skipped: …" note |
| C, Dense table, Skeleton preview | compact selects on the left, the Skeleton's file tree and Configuration file on the right | selectable with ⚠, reported as an error | greyed, "n/a, needs …" |

## What all variants share

- **State lives in the URL.** Only the module path and the Options you touched are in it (`x.<option>=<choice>`). Untouched Options keep following their defaults, including conditional ones: pick PostgreSQL and `container` turns to Docker. A URL is a shareable, reload-stable Configuration.
- **"Back to default"** removes an Option from the URL. In C the select's first entry, "default (X)", does the same thing.
- **Remembered Choices.** An explicit Choice on an Option that becomes inapplicable stays in the URL, is ignored, and comes back when the Option applies again.
- **Downloads.** `<name>.zip` holds placeholder files plus the real `goskel.toml`. "Configuration file" downloads `goskel.toml` on its own. The zip is refused while there are errors; the Configuration file never is.
- **Loading a file.** "Load a Configuration file" in the header uploads a `.toml` and redirects to its URL state.
- **CLI parity.** Every variant shows the minimal equivalent `goskel new …` command.
- **`serve`.** `--addr` defaults to `127.0.0.1:7070` (`0.0.0.0` inside a container), and `--open` launches the browser. A short banner prints to stderr. Nothing is written to disk.

## Not real

`goskel` is a placeholder name, since Naming is still fog. The catalogue is hard-coded core-tier Go, there is no engine, the TOML parser is a toy, and htmx loads from a CDN. Presets are left out because their ticket is still open.
