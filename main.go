package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	// Initialize model
	m, err := initialModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing KubeDoctor: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure:\n")
		fmt.Fprintf(os.Stderr, "  - kubectl is configured (run 'kubectl cluster-info')\n")
		fmt.Fprintf(os.Stderr, "  - You have access to a Kubernetes cluster\n")
		fmt.Fprintf(os.Stderr, "  - Your kubeconfig file is valid (~/.kube/config)\n")
		os.Exit(1)
	}

	// Create Bubbletea program
	p := tea.NewProgram(
		m,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running KubeDoctor: %v\n", err)
		os.Exit(1)
	}
}
