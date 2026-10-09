package main

import (
	"net/url"
	"strings"
)

// explicitPrefix marks hidden state fields. Visible controls belong to a form
// that never submits, so the URL carries only the Choices the user touched.
const explicitPrefix = "x."

func stateFromQuery(query url.Values) State {
	state := State{Module: strings.TrimSpace(query.Get("module")), Explicit: map[string][]string{}}
	for key, values := range query {
		optionID, ok := strings.CutPrefix(key, explicitPrefix)
		if ok && optionByID(optionID) != nil {
			state.Explicit[optionID] = splitChoices(values[0])
		}
	}
	return state
}

func splitChoices(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func (s State) Query() url.Values {
	query := url.Values{}
	query.Set("module", s.Module)
	for optionID, choices := range s.Explicit {
		query.Set(explicitPrefix+optionID, strings.Join(choices, ","))
	}
	return query
}

type HiddenField struct {
	Name  string
	Value string
}

func (s State) HiddenFields() []HiddenField {
	var fields []HiddenField
	for _, group := range catalogue {
		for _, option := range group.Options {
			if choices, ok := s.Explicit[option.ID]; ok {
				fields = append(fields, HiddenField{Name: explicitPrefix + option.ID, Value: strings.Join(choices, ",")})
			}
		}
	}
	return fields
}

// stateFromConfigurationFile reads just enough TOML for the prototype:
// module = "..." and key = "v" or key = ["a", "b"] lines under [choices].
func stateFromConfigurationFile(content string) State {
	state := State{Explicit: map[string][]string{}}
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key = strings.TrimSpace(key)
		values := parseTOMLStrings(value)
		switch {
		case key == "module" && len(values) == 1:
			state.Module = values[0]
		case optionByID(key) != nil:
			state.Explicit[key] = values
		}
	}
	return state
}

func parseTOMLStrings(value string) []string {
	value = strings.Trim(strings.TrimSpace(value), "[]")
	values := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.Trim(strings.TrimSpace(part), `"`); part != "" {
			values = append(values, part)
		}
	}
	return values
}
