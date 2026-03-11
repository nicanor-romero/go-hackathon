package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type Team struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TeamMember struct {
	Login string `json:"login"`
}

type teamsFetchedMsg struct {
	teams []Team
	err   error
}

type membersFetchedMsg struct {
	members []TeamMember
	err     error
}

func filterTeams(teams []Team, query string) []Team {
	if query == "" {
		return teams
	}
	q := strings.ToLower(query)
	var out []Team
	for _, t := range teams {
		if strings.Contains(strings.ToLower(t.Name), q) ||
			strings.Contains(strings.ToLower(t.Slug), q) {
			out = append(out, t)
		}
	}
	return out
}

func fetchTeamsCmd(org string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "api", fmt.Sprintf("/orgs/%s/teams", org), "--paginate")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return teamsFetchedMsg{err: fmt.Errorf("gh api teams: %s", msg)}
		}
		var teams []Team
		if err := json.Unmarshal(out, &teams); err != nil {
			return teamsFetchedMsg{err: fmt.Errorf("parse teams: %w", err)}
		}
		return teamsFetchedMsg{teams: teams}
	}
}

func fetchMembersCmd(org, slug string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("gh", "api", fmt.Sprintf("/orgs/%s/teams/%s/members", org, slug), "--paginate")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return membersFetchedMsg{err: fmt.Errorf("gh api members: %s", msg)}
		}
		var members []TeamMember
		if err := json.Unmarshal(out, &members); err != nil {
			return membersFetchedMsg{err: fmt.Errorf("parse members: %w", err)}
		}
		return membersFetchedMsg{members: members}
	}
}
