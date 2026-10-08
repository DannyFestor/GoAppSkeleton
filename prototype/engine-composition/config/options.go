package config

import (
	"errors"
	"fmt"
	"slices"
)

type Option struct {
	Name    string
	Choices []string
	Default string
}

type Selection struct {
	Option string
	Choice string
}

type RelationKind string

const (
	Requires      RelationKind = "requires"
	ConflictsWith RelationKind = "conflicts with"
)

// Relation reads "When <Kind> Other"; Requires with Other.Choice == "" means "any Choice except none".
type Relation struct {
	When  Selection
	Kind  RelationKind
	Other Selection
}

// Catalogue order is also render order: slot contributions are sorted by it, so output never depends on map iteration.
var Catalogue = []Option{
	{Name: "router", Choices: []string{"stdlib", "chi"}, Default: "stdlib"},
	{Name: "database", Choices: []string{None, "postgres", "sqlite"}, Default: None},
	{Name: "sessions", Choices: []string{None, "cookie", "database"}, Default: None},
	{Name: "cache", Choices: []string{None, "redis"}, Default: None},
}

var Relations = []Relation{
	{When: Selection{"sessions", "database"}, Kind: Requires, Other: Selection{"database", ""}},
}

func New(module string, choices map[string]string) (Configuration, error) {
	cfg := Configuration{Module: module, Choices: withDefaults(choices)}
	if err := validate(cfg); err != nil {
		return Configuration{}, err
	}
	return cfg, nil
}

func withDefaults(choices map[string]string) map[string]string {
	complete := make(map[string]string, len(Catalogue))
	for _, option := range Catalogue {
		complete[option.Name] = option.Default
		if choice, ok := choices[option.Name]; ok {
			complete[option.Name] = choice
		}
	}
	return complete
}

func validate(cfg Configuration) error {
	var errs []error
	for _, option := range Catalogue {
		if !slices.Contains(option.Choices, cfg.Choice(option.Name)) {
			errs = append(errs, fmt.Errorf("%s: unknown choice %q", option.Name, cfg.Choice(option.Name)))
		}
	}
	for _, relation := range Relations {
		if cfg.Is(relation.When.Option, relation.When.Choice) && !relation.holds(cfg) {
			errs = append(errs, fmt.Errorf("%s=%s %s %s", relation.When.Option, relation.When.Choice, relation.Kind, relation.Other.describe()))
		}
	}
	return errors.Join(errs...)
}

func (r Relation) holds(cfg Configuration) bool {
	matches := r.Other.matches(cfg)
	if r.Kind == ConflictsWith {
		return !matches
	}
	return matches
}

func (s Selection) matches(cfg Configuration) bool {
	if s.Choice == "" {
		return cfg.Uses(s.Option)
	}
	return cfg.Is(s.Option, s.Choice)
}

func (s Selection) describe() string {
	if s.Choice == "" {
		return s.Option
	}
	return s.Option + "=" + s.Choice
}

// All enumerates every valid Configuration, the input to the cross-style comparison and build tests.
func All(module string) []Configuration {
	var all []Configuration
	for _, choices := range product(Catalogue) {
		if cfg, err := New(module, choices); err == nil {
			all = append(all, cfg)
		}
	}
	return all
}

func product(options []Option) []map[string]string {
	if len(options) == 0 {
		return []map[string]string{{}}
	}
	var combinations []map[string]string
	for _, rest := range product(options[1:]) {
		for _, choice := range options[0].Choices {
			combination := map[string]string{options[0].Name: choice}
			for k, v := range rest {
				combination[k] = v
			}
			combinations = append(combinations, combination)
		}
	}
	return combinations
}

func (c Configuration) Name() string {
	name := ""
	for _, option := range Catalogue {
		name += option.Name + "=" + c.Choice(option.Name) + ","
	}
	return name[:len(name)-1]
}
