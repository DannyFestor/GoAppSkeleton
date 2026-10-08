package refpatch

import (
	"example.com/enginecomp/config"
)

type patch struct {
	when  []config.Selection
	apply func(files goFiles) error
}

// Order is not catalogue order: router/chi rewrites every route registration, including those other patches add,
// so it must run last; sqlite placeholders rewrite a file that sessions/database adds, so they follow it.
var patches = []patch{
	{when: []config.Selection{{Option: "database", Choice: "postgres"}}, apply: addDatabase("postgres")},
	{when: []config.Selection{{Option: "database", Choice: "sqlite"}}, apply: addDatabase("sqlite")},
	{when: []config.Selection{{Option: "sessions", Choice: "cookie"}}, apply: addCookieSessions},
	{when: []config.Selection{{Option: "sessions", Choice: "database"}}, apply: addSQLSessions},
	{when: []config.Selection{{Option: "sessions", Choice: "database"}, {Option: "database", Choice: "sqlite"}}, apply: useSQLitePlaceholders},
	{when: []config.Selection{{Option: "cache", Choice: "redis"}}, apply: addCache},
	{when: []config.Selection{{Option: "router", Choice: "chi"}}, apply: useChiRouter},
}

func applyPatches(files goFiles, cfg config.Configuration) error {
	for _, p := range patches {
		if !p.appliesTo(cfg) {
			continue
		}
		if err := p.apply(files); err != nil {
			return err
		}
	}
	return nil
}

func (p patch) appliesTo(cfg config.Configuration) bool {
	for _, selection := range p.when {
		if !cfg.Is(selection.Option, selection.Choice) {
			return false
		}
	}
	return true
}

func addOverlay(files goFiles, dir string) error {
	return files.addFrom(mustSub(goSources, "testdata/overlays/"+dir))
}
