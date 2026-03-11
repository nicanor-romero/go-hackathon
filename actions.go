package main

import (
	"fmt"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

type reviewMode int

const (
	reviewModeApprove reviewMode = iota
	reviewModeRequestChanges
)

type actionDoneMsg struct {
	action string
	err    error
}

func openBrowserCmd(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command(
			"gh", "pr", "view",
			fmt.Sprintf("%d", number),
			"--repo", repo,
			"--web",
		).Run()
		return actionDoneMsg{action: "browser", err: err}
	}
}

func reviewCmd(number int, repo string, mode reviewMode, body string) tea.Cmd {
	return func() tea.Msg {
		args := []string{
			"pr", "review",
			fmt.Sprintf("%d", number),
			"--repo", repo,
		}
		if mode == reviewModeApprove {
			args = append(args, "--approve")
		} else {
			args = append(args, "--request-changes")
		}
		if body != "" {
			args = append(args, "-b", body)
		}
		err := exec.Command("gh", args...).Run()
		return actionDoneMsg{action: "review", err: err}
	}
}

func checkoutCmd(number int, repo string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command(
			"gh", "pr", "checkout",
			fmt.Sprintf("%d", number),
			"--repo", repo,
		).Run()
		return actionDoneMsg{action: "checkout", err: err}
	}
}
