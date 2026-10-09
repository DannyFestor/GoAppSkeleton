package main

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
)

// binName is a placeholder: the map's Naming fog has not settled it.
const binName = "goskel"
const generatorVersion = "v0.0.0-prototype"

func (r *Resolved) ProjectName() string {
	module := strings.TrimRight(r.State.Module, "/")
	if module == "" {
		return "app"
	}
	return path.Base(module)
}

func (r *Resolved) ConfigurationFileName() string {
	return binName + ".toml"
}

func (r *Resolved) ConfigurationTOML() string {
	var b strings.Builder
	fmt.Fprintf(&b, "generator = %q\nmodule = %q\n\n[choices]\n", generatorVersion, r.State.Module)
	for _, group := range catalogue {
		fmt.Fprintf(&b, "# %s\n", group.Label)
		for _, option := range group.Options {
			if r.Applicable[option.ID] {
				fmt.Fprintf(&b, "%s = %s\n", option.ID, tomlValue(option, r.Choices[option.ID]))
			}
		}
	}
	return b.String()
}

func tomlValue(option *Option, choices []string) string {
	if option.Kind != Multi {
		return fmt.Sprintf("%q", choices[0])
	}
	quoted := make([]string, len(choices))
	for i, choice := range choices {
		quoted[i] = fmt.Sprintf("%q", choice)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func (r *Resolved) EquivalentCommand() string {
	parts := []string{binName, "new", r.State.Module}
	for _, group := range catalogue {
		for _, option := range group.Options {
			if r.Applicable[option.ID] && r.DiffersFromDefault(option) {
				parts = append(parts, flagFor(option, r.Choices[option.ID]))
			}
		}
	}
	return strings.Join(parts, " ")
}

func flagFor(option *Option, choices []string) string {
	if option.Kind == Toggle {
		if choices[0] == "on" {
			return "--" + option.ID
		}
		return "--" + option.ID + "=false"
	}
	return "--" + option.ID + "=" + strings.Join(choices, ",")
}

func (r *Resolved) Files() []string {
	files := slices.Clone(baseFiles)
	files = append(files, r.ConfigurationFileName())
	for _, group := range catalogue {
		for _, option := range group.Options {
			files = append(files, r.filesOf(option)...)
		}
	}
	slices.Sort(files)
	return slices.Compact(files)
}

func (r *Resolved) filesOf(option *Option) []string {
	var files []string
	for _, choiceID := range r.Choices[option.ID] {
		if choice := option.choiceByID(choiceID); choice != nil {
			files = append(files, choice.Files...)
		}
	}
	return files
}

// WriteZip stands in for the engine: every file is a placeholder except the
// Configuration file, which is real.
func (r *Resolved) WriteZip(w io.Writer) error {
	archive := zip.NewWriter(w)
	root := r.ProjectName() + "/"
	for _, file := range r.Files() {
		entry, err := archive.Create(root + file)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(entry, r.contentOf(file)); err != nil {
			return err
		}
	}
	return archive.Close()
}

func (r *Resolved) contentOf(file string) string {
	if file == r.ConfigurationFileName() {
		return r.ConfigurationTOML()
	}
	return "PROTOTYPE placeholder: the engine would render " + file + "\n"
}
