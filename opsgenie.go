package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// httpClient that does not follow redirects (OpsGenie redirects to HTML login on bad auth)
var httpClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func fetchIncidents(apiKey, apiURL string) ([]Incident, error) {
	url := apiURL + "/alerts?query=status:open&limit=100&order=desc"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "GenieKey "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching alerts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpsGenie API returned %d: %s", resp.StatusCode, string(body))
	}

	// Guard against non-JSON responses (e.g. HTML error pages from proxies)
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return nil, fmt.Errorf("OpsGenie returned unexpected content-type %q (status %d): %.200s", ct, resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID          string            `json:"id"`
			Message     string            `json:"message"`
			Priority    string            `json:"priority"`
			Description string            `json:"description"`
			CreatedAt   time.Time         `json:"createdAt"`
			Details     map[string]string `json:"details"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var incidents []Incident
	for _, a := range result.Data {
		inc := Incident{
			ID:          a.ID,
			Title:       a.Message,
			Priority:    a.Priority,
			Description: a.Description,
			StartTime:   a.CreatedAt,
			RawDetails:  a.Details,
		}
		// Extract namespace, deployment, team from details (case-insensitive)
		for k, v := range a.Details {
			switch strings.ToLower(k) {
			case "namespace":
				inc.Namespace = v
			case "deployment":
				inc.Deployment = v
			case "team":
				inc.Team = v
			}
		}
		incidents = append(incidents, inc)
	}

	// Sort by priority (P1 first), then by start time
	priorityOrder := map[string]int{"P1": 0, "P2": 1, "P3": 2, "P4": 3, "P5": 4}
	sort.Slice(incidents, func(i, j int) bool {
		pi := priorityOrder[incidents[i].Priority]
		pj := priorityOrder[incidents[j].Priority]
		if pi != pj {
			return pi < pj
		}
		return incidents[i].StartTime.Before(incidents[j].StartTime)
	})

	return incidents, nil
}

func escalateAlert(apiKey, apiURL string, incident Incident, targetTeam string) error {
	payload := map[string]interface{}{
		"message":     fmt.Sprintf("[Escalated] %s", incident.Title),
		"priority":    incident.Priority,
		"description": fmt.Sprintf("Escalated from incident %s: %s", incident.ID, incident.Description),
		"responders": []map[string]string{
			{"type": "team", "name": targetTeam},
		},
		"details": incident.RawDetails,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL+"/alerts", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "GenieKey "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending escalation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpsGenie API returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
