// Package config is the part every style shares: the Option declaration and a validated Configuration.
package config

const None = "none"

type Configuration struct {
	Module  string
	Choices map[string]string
}

func (c Configuration) Is(option, choice string) bool {
	return c.Choices[option] == choice
}

func (c Configuration) Uses(option string) bool {
	choice, ok := c.Choices[option]
	return ok && choice != None
}

func (c Configuration) Choice(option string) string {
	return c.Choices[option]
}
