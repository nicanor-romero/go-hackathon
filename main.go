package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func main() {
	var repo string
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--repo" && i+1 < len(args) {
			repo = args[i+1]
		} else if strings.HasPrefix(arg, "--repo=") {
			repo = strings.TrimPrefix(arg, "--repo=")
		}
	}

	if repo == "" {
		fmt.Fprintln(os.Stderr, "Usage: pr-tui --repo owner/name")
		os.Exit(1)
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		fmt.Fprintf(os.Stderr, "Error: --repo must be in 'owner/name' format, got: %q\n", repo)
		os.Exit(1)
	}

	org := parts[0]
	m := newAppModel(repo, org)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
