package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(
		initialShieldModel(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
