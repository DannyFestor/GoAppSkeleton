package main

import (
	"fmt"
	"slices"
	"strings"
)

type Source string

const (
	FromUser         Source = "you"
	FromDefault      Source = "default"
	FromInapplicable Source = "not applicable"
)

// State is everything the browser sends back: the module path and only the
// Options the user touched, so untouched ones keep following their defaults.
type State struct {
	Module   string
	Explicit map[string][]string
}

type Resolved struct {
	State      State
	Choices    map[string][]string
	Sources    map[string]Source
	Applicable map[string]bool
	Blocked    map[string]map[string]string
	Errors     []string
}

func Resolve(state State) *Resolved {
	r := &Resolved{
		State:      state,
		Choices:    map[string][]string{},
		Sources:    map[string]Source{},
		Applicable: map[string]bool{},
		Blocked:    map[string]map[string]string{},
	}
	for _, group := range catalogue {
		for _, option := range group.Options {
			r.resolveOption(option)
		}
	}
	for _, group := range catalogue {
		for _, option := range group.Options {
			r.checkRequires(option)
		}
	}
	r.checkModule()
	return r
}

func (r *Resolved) checkModule() {
	switch {
	case r.State.Module == "":
		r.Errors = append(r.Errors, "module path is required, for example github.com/acme/shop")
	case strings.ContainsAny(r.State.Module, " \t"):
		r.Errors = append(r.Errors, "module path cannot contain spaces")
	}
}

func (r *Resolved) resolveOption(option *Option) {
	r.Applicable[option.ID] = r.allMatch(option.AppliesWhen)
	switch {
	case !r.Applicable[option.ID]:
		r.Choices[option.ID] = absent(option)
		r.Sources[option.ID] = FromInapplicable
	case r.isExplicit(option.ID):
		r.Choices[option.ID] = r.State.Explicit[option.ID]
		r.Sources[option.ID] = FromUser
	default:
		r.Choices[option.ID] = r.defaultFor(option)
		r.Sources[option.ID] = FromDefault
	}
}

func (r *Resolved) isExplicit(optionID string) bool {
	_, ok := r.State.Explicit[optionID]
	return ok
}

func absent(option *Option) []string {
	if option.Kind == Multi {
		return []string{}
	}
	return []string{"none"}
}

func (r *Resolved) defaultFor(option *Option) []string {
	for _, conditional := range option.ConditionalDefaults {
		if r.matches(conditional.When) {
			return []string{conditional.Choice}
		}
	}
	return option.Default
}

func (r *Resolved) checkRequires(option *Option) {
	if !r.Applicable[option.ID] {
		return
	}
	r.Blocked[option.ID] = map[string]string{}
	for _, choice := range option.Choices {
		unmet := r.firstUnmet(choice.Requires)
		if unmet == nil {
			continue
		}
		r.Blocked[option.ID][choice.ID] = "needs " + describeCond(*unmet)
		if r.IsSelected(option.ID, choice.ID) {
			r.Errors = append(r.Errors, r.requiresError(option, choice, *unmet))
		}
	}
}

func (r *Resolved) requiresError(option *Option, choice Choice, unmet Cond) string {
	other := optionByID(unmet.Option)
	return fmt.Sprintf("%s=%s requires %s, but %s=%s (%s). Fix: change %s, or choose another %s.",
		option.ID, choice.ID, describeCond(unmet),
		other.ID, strings.Join(r.Choices[other.ID], ","), sourceInError(r.Sources[other.ID]),
		other.Label, option.Label)
}

func sourceInError(source Source) string {
	if source == FromUser {
		return "your choice"
	}
	return string(source)
}

func (r *Resolved) firstUnmet(conds []Cond) *Cond {
	for i := range conds {
		if !r.matches(conds[i]) {
			return &conds[i]
		}
	}
	return nil
}

func (r *Resolved) allMatch(conds []Cond) bool {
	return r.firstUnmet(conds) == nil
}

func (r *Resolved) matches(cond Cond) bool {
	selected := r.Choices[cond.Option]
	if len(cond.Choices) == 0 {
		return slices.ContainsFunc(selected, func(id string) bool { return id != "none" })
	}
	return slices.ContainsFunc(selected, func(id string) bool { return slices.Contains(cond.Choices, id) })
}

func describeCond(cond Cond) string {
	option := optionByID(cond.Option)
	if len(cond.Choices) == 0 {
		return option.ID + " ≠ none"
	}
	if option.Kind == Multi {
		return option.ID + " contains " + strings.Join(cond.Choices, " or ")
	}
	return option.ID + " = " + strings.Join(cond.Choices, " or ")
}

func (r *Resolved) IsSelected(optionID, choiceID string) bool {
	return slices.Contains(r.Choices[optionID], choiceID)
}

// AppliesBecause explains a hidden Option, for variants that mention it.
func AppliesBecause(option *Option) string {
	descriptions := make([]string, len(option.AppliesWhen))
	for i, cond := range option.AppliesWhen {
		descriptions[i] = describeCond(cond)
	}
	return strings.Join(descriptions, " and ")
}

// DiffersFromDefault drives the equivalent command: only Choices a default
// would not have produced need a flag.
func (r *Resolved) DiffersFromDefault(option *Option) bool {
	return r.Sources[option.ID] == FromUser && !slices.Equal(r.Choices[option.ID], r.defaultFor(option))
}
