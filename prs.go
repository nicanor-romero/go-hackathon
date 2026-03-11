package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

type CIStatus int

const (
	CIUnknown CIStatus = iota
	CIPending
	CIPass
	CIFail
)

type PR struct {
	Number      int
	Title       string
	AuthorLogin string
	IsDraft     bool
	Mergeable   string // "MERGEABLE" | "CONFLICTING" | "UNKNOWN"
	CIStatus    CIStatus
	UpdatedAt   time.Time
	HeadRefName string
}

type prsFetchedMsg struct {
	prs []PR
	err error
}

type ghCheckRun struct {
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
	State      string `json:"state"`
}

type ghPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	IsDraft           bool         `json:"isDraft"`
	Mergeable         string       `json:"mergeable"`
	StatusCheckRollup []ghCheckRun `json:"statusCheckRollup"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	HeadRefName       string       `json:"headRefName"`
}

func computeCIStatus(checks []ghCheckRun) CIStatus {
	if len(checks) == 0 {
		return CIUnknown
	}
	hasFailure := false
	hasPending := false
	allSuccess := true
	for _, c := range checks {
		switch c.Conclusion {
		case "FAILURE", "ERROR":
			hasFailure = true
			allSuccess = false
		case "SUCCESS", "SKIPPED":
			// ok
		default:
			allSuccess = false
		}
		switch c.Status {
		case "IN_PROGRESS", "QUEUED":
			hasPending = true
			allSuccess = false
		}
	}
	if hasFailure {
		return CIFail
	}
	if hasPending {
		return CIPending
	}
	if allSuccess {
		return CIPass
	}
	return CIUnknown
}

// isTransientErr returns true for GitHub API errors that are safe to retry.
func isTransientErr(stderr string) bool {
	for _, code := range []string{"502", "503", "504", "429"} {
		if strings.Contains(stderr, "HTTP "+code) {
			return true
		}
	}
	return false
}

func fetchPRsCmd(repo string) tea.Cmd {
	return func() tea.Msg {
		const maxAttempts = 3
		delays := []time.Duration{2 * time.Second, 5 * time.Second}

		var lastErr string
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if attempt > 0 {
				time.Sleep(delays[attempt-1])
			}
			cmd := exec.Command(
				"gh", "pr", "list",
				"--repo", repo,
				"--json", "number,title,author,isDraft,mergeable,statusCheckRollup,updatedAt,headRefName",
				"--limit", "100",
			)
			var stderr strings.Builder
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil {
				lastErr = strings.TrimSpace(stderr.String())
				if lastErr == "" {
					lastErr = err.Error()
				}
				if isTransientErr(lastErr) {
					continue // retry
				}
				return prsFetchedMsg{err: fmt.Errorf("gh pr list: %s", lastErr)}
			}

			var raw []ghPR
			if err := json.Unmarshal(out, &raw); err != nil {
				return prsFetchedMsg{err: fmt.Errorf("parse prs: %w", err)}
			}
			prs := make([]PR, 0, len(raw))
			for _, g := range raw {
				prs = append(prs, PR{
					Number:      g.Number,
					Title:       g.Title,
					AuthorLogin: g.Author.Login,
					IsDraft:     g.IsDraft,
					Mergeable:   g.Mergeable,
					CIStatus:    computeCIStatus(g.StatusCheckRollup),
					UpdatedAt:   g.UpdatedAt,
					HeadRefName: g.HeadRefName,
				})
			}
			return prsFetchedMsg{prs: prs}
		}
		return prsFetchedMsg{err: fmt.Errorf("gh pr list: %s (after %d attempts)", lastErr, maxAttempts)}
	}
}

func filterAndSortPRs(prs []PR, teamMembers map[string]bool) []PR {
	var filtered []PR
	for _, pr := range prs {
		if teamMembers[pr.AuthorLogin] {
			filtered = append(filtered, pr)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})
	return filtered
}
