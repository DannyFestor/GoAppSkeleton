// Command render prints or writes one Skeleton: go run ./cmd/render -style slots -set router=chi -set database=postgres
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"example.com/enginecomp/config"
	"example.com/enginecomp/styles"
)

type choiceFlags map[string]string

func (c choiceFlags) String() string { return fmt.Sprint(map[string]string(c)) }

func (c choiceFlags) Set(value string) error {
	option, choice, ok := strings.Cut(value, "=")
	if !ok {
		return fmt.Errorf("want option=choice, got %q", value)
	}
	c[option] = choice
	return nil
}

func main() {
	choices := choiceFlags{}
	style := flag.String("style", styles.Control, "wholefile, slots, overlay or refpatch")
	out := flag.String("out", "", "write the Skeleton here instead of printing it")
	flag.Var(choices, "set", "option=choice, repeatable")
	flag.Parse()

	if err := run(*style, *out, choices); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(style, out string, choices map[string]string) error {
	render, ok := styles.All[style]
	if !ok {
		return fmt.Errorf("unknown style %q", style)
	}
	cfg, err := config.New("example.com/myapp", choices)
	if err != nil {
		return err
	}
	tree, err := render(cfg)
	if err != nil {
		return err
	}
	if out != "" {
		return tree.WriteTo(out)
	}
	for _, path := range tree.Paths() {
		fmt.Printf("===== %s\n%s\n", path, tree[path])
	}
	return nil
}
