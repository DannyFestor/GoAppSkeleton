package slots

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Route is what a route contribution hands to the router Choice's "router.route" provider.
type Route struct {
	Method  string
	Path    string
	Handler string
}

func (r *renderer) funcs() template.FuncMap {
	return template.FuncMap{
		"lines":  r.joined("\n"),
		"blocks": r.joined("\n\n"),
		"words":  r.joined(" "),
		"wrap":   r.wrap,
		"use":    r.use,
		"route":  r.route,
		"pascal": pascal,
	}
}

func (r *renderer) joined(separator string) func(slot string) (string, error) {
	return func(slot string) (string, error) {
		contributions, err := r.contributions(slot, r.cfg)
		return strings.Join(contributions, separator), err
	}
}

// wrap folds every contribution around the previous expression, so middleware nests in catalogue order.
func (r *renderer) wrap(slot, inner string) (string, error) {
	expression := inner
	for _, set := range r.setsDefining(slot) {
		wrapped, err := execute(set.tmpl, slot, expression)
		if err != nil {
			return "", err
		}
		expression = wrapped
	}
	return expression, nil
}

// use resolves a provider: exactly one selected fragment set must define it, unlike a slot which takes any number.
func (r *renderer) use(name string, args ...any) (string, error) {
	providers := r.setsDefining(name)
	if len(providers) != 1 {
		return "", fmt.Errorf("%s: want exactly one provider, got %d", name, len(providers))
	}
	var dot any = r.cfg
	if len(args) == 1 {
		dot = args[0]
	}
	return execute(providers[0].tmpl, name, dot)
}

func (r *renderer) route(method, path, handler string) (string, error) {
	return r.use("router.route", Route{Method: method, Path: path, Handler: handler})
}

func (r *renderer) contributions(slot string, dot any) ([]string, error) {
	var contributions []string
	for _, set := range r.setsDefining(slot) {
		contribution, err := execute(set.tmpl, slot, dot)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, contribution)
	}
	return contributions, nil
}

func (r *renderer) setsDefining(name string) []fragmentSet {
	var defining []fragmentSet
	for _, set := range r.sets {
		if set.tmpl.Lookup(name) != nil {
			defining = append(defining, set)
		}
	}
	return defining
}

// Only newlines are trimmed: a leading tab is meaningful in Makefile recipes and go.mod require lines.
func execute(tmpl *template.Template, name string, dot any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, dot); err != nil {
		return "", err
	}
	return strings.Trim(buf.String(), "\n"), nil
}

func pascal(s string) string {
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}
