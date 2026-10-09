package main

import (
	"html/template"
	"slices"
	"strconv"
	"strings"
)

type Variant struct {
	Key  string
	Name string
}

var variants = []Variant{
	{Key: "A", Name: "Long form, live sidebar"},
	{Key: "B", Name: "Stepper, mirrors the wizard"},
	{Key: "C", Name: "Dense table, Skeleton preview"},
}

type Page struct {
	*Resolved
	Variant  Variant
	Previous Variant
	Next     Variant
	Step     int
	Groups   []Group
}

func newPage(resolved *Resolved, variantKey string, step int) Page {
	index := max(0, slices.IndexFunc(variants, func(v Variant) bool { return v.Key == variantKey }))
	return Page{
		Resolved: resolved,
		Variant:  variants[index],
		Previous: variants[(index+len(variants)-1)%len(variants)],
		Next:     variants[(index+1)%len(variants)],
		Step:     min(max(step, 0), len(catalogue)+1),
		Groups:   catalogue,
	}
}

func (p Page) URL(variantKey string, step int) template.URL {
	query := p.State.Query()
	query.Set("variant", variantKey)
	query.Set("step", strconv.Itoa(step))
	return template.URL("/?" + query.Encode())
}

func (p Page) ZipURL() template.URL {
	return template.URL("/skeleton.zip?" + p.State.Query().Encode())
}

func (p Page) ConfigurationURL() template.URL {
	return template.URL("/configuration.toml?" + p.State.Query().Encode())
}

func (p Page) ReviewStep() int {
	return len(catalogue) + 1
}

func (p Page) StepGroup() Group {
	return catalogue[p.Step-1]
}

func (r *Resolved) Selected(optionID, choiceID string) bool {
	return r.IsSelected(optionID, choiceID)
}

func (r *Resolved) BlockedReason(optionID, choiceID string) string {
	return r.Blocked[optionID][choiceID]
}

func (r *Resolved) Source(optionID string) Source {
	return r.Sources[optionID]
}

func (r *Resolved) IsInapplicable(optionID string) bool {
	return !r.Applicable[optionID]
}

func (r *Resolved) DefaultLabel(option *Option) string {
	return labelsOf(option, r.defaultFor(option))
}

func (r *Resolved) ChoiceLabels(option *Option) string {
	return labelsOf(option, r.Choices[option.ID])
}

func labelsOf(option *Option, choiceIDs []string) string {
	if len(choiceIDs) == 0 {
		return "nothing"
	}
	labels := make([]string, len(choiceIDs))
	for i, id := range choiceIDs {
		labels[i] = option.choiceByID(id).Label
	}
	return strings.Join(labels, ", ")
}

func (r *Resolved) HiddenNotes(group Group) []string {
	var notes []string
	for _, option := range group.Options {
		if !r.Applicable[option.ID] {
			notes = append(notes, option.Label+" (needs "+AppliesBecause(option)+")")
		}
	}
	return notes
}

func (r *Resolved) LeftOutNotes(option *Option) []string {
	var notes []string
	for _, choice := range option.Choices {
		if reason := r.BlockedReason(option.ID, choice.ID); reason != "" && !r.IsSelected(option.ID, choice.ID) {
			notes = append(notes, choice.Label+" ("+reason+")")
		}
	}
	return notes
}

// GroupSummary feeds the stepper's "keep defaults" line for a group.
func (r *Resolved) GroupSummary(group Group) string {
	var parts []string
	for _, option := range group.Options {
		if r.Applicable[option.ID] {
			parts = append(parts, option.Label+": "+r.ChoiceLabels(option))
		}
	}
	return strings.Join(parts, " · ")
}

type TreeLine struct {
	Depth int
	Name  string
	IsDir bool
	Added bool
}

// Tree marks files that the all-defaults Configuration would not produce.
func (r *Resolved) Tree() []TreeLine {
	defaults := Resolve(State{Module: r.State.Module}).Files()
	var lines []TreeLine
	seenDirs := map[string]bool{}
	for _, file := range r.Files() {
		parts := strings.Split(file, "/")
		for depth := range len(parts) - 1 {
			dir := strings.Join(parts[:depth+1], "/")
			if !seenDirs[dir] {
				seenDirs[dir] = true
				lines = append(lines, TreeLine{Depth: depth, Name: parts[depth] + "/", IsDir: true})
			}
		}
		lines = append(lines, TreeLine{Depth: len(parts) - 1, Name: parts[len(parts)-1], Added: !slices.Contains(defaults, file)})
	}
	return lines
}
