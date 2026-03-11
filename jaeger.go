package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const jaegerURL = "https://tools.masstack.com/tracing/api"

// JaegerClient handles Jaeger API queries
type JaegerClient struct {
	baseURL    string
	authCookie string
	client     *http.Client
}

// Dependency represents a service dependency
type Dependency struct {
	Parent    string `json:"parent"`
	Child     string `json:"child"`
	CallCount int64  `json:"callCount"`
}

// ServiceDependencies holds dependencies for a service
type ServiceDependencies struct {
	Service      string
	Dependencies []string
	Parents      []string
}

// NewJaegerClient creates a new Jaeger client
func NewJaegerClient(baseURL, authCookie string) *JaegerClient {
	return &JaegerClient{
		baseURL:    baseURL,
		authCookie: authCookie,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetServices fetches all available services
func (j *JaegerClient) GetServices(ctx context.Context) ([]string, error) {
	u := fmt.Sprintf("%s/services", j.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	if j.authCookie != "" {
		req.Header.Set("Cookie", j.authCookie)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jaeger returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetDependencies fetches service dependencies
func (j *JaegerClient) GetDependencies(ctx context.Context) ([]Dependency, error) {
	// Get dependencies for last 24 hours
	endTs := time.Now().Unix() * 1000
	lookback := int64(24 * 60 * 60 * 1000) // 24 hours in milliseconds

	u := fmt.Sprintf("%s/dependencies?endTs=%d&lookback=%d", j.baseURL, endTs, lookback)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	if j.authCookie != "" {
		req.Header.Set("Cookie", j.authCookie)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jaeger returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []Dependency `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetServiceDependencies returns first-level dependencies for a service
func (j *JaegerClient) GetServiceDependencies(ctx context.Context, serviceName string) (ServiceDependencies, error) {
	deps, err := j.GetDependencies(ctx)
	if err != nil {
		return ServiceDependencies{}, err
	}

	result := ServiceDependencies{
		Service:      serviceName,
		Dependencies: []string{},
		Parents:      []string{},
	}

	seen := make(map[string]bool)

	for _, dep := range deps {
		// Find children (services this service depends on)
		if dep.Parent == serviceName {
			if !seen[dep.Child] {
				result.Dependencies = append(result.Dependencies, dep.Child)
				seen[dep.Child] = true
			}
		}

		// Find parents (services that depend on this service)
		if dep.Child == serviceName {
			if !seen[dep.Parent] {
				result.Parents = append(result.Parents, dep.Parent)
				seen[dep.Parent] = true
			}
		}
	}

	return result, nil
}
