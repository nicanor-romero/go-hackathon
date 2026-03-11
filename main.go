package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	apiKey := os.Getenv("OPSGENIE_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OPSGENIE_API_KEY environment variable is required")
		os.Exit(1)
	}

	slackToken := os.Getenv("SLACK_BOT_TOKEN")
	if slackToken == "" {
		fmt.Fprintln(os.Stderr, "Error: SLACK_BOT_TOKEN environment variable is required")
		os.Exit(1)
	}

	apiURL := os.Getenv("OPSGENIE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.eu.opsgenie.com/v2"
	}

	p := tea.NewProgram(
		initialWarModel(apiKey, apiURL, slackToken),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
