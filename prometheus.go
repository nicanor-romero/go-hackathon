package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const prometheusURL = "https://prometheus-mas-billing-prod.private-eusw1.prod.k8s.masmovil.com"

// PrometheusClient handles Prometheus API queries
type PrometheusClient struct {
	baseURL string
	client  *http.Client
}

// MetricMetadata represents a Prometheus metric
type MetricMetadata struct {
	Name string
	Type string
	Help string
}

// QueryResult represents a Prometheus query result
type QueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetMetricNames fetches all available metric names
func (p *PrometheusClient) GetMetricNames(ctx context.Context) ([]string, error) {
	u := fmt.Sprintf("%s/api/v1/label/__name__/values", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", result.Status)
	}

	return result.Data, nil
}

// QueryInstant performs an instant query
func (p *PrometheusClient) QueryInstant(ctx context.Context, query string) ([]QueryResult, error) {
	u := fmt.Sprintf("%s/api/v1/query", p.baseURL)

	params := url.Values{}
	params.Add("query", query)

	req, err := http.NewRequestWithContext(ctx, "GET", u+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string        `json:"resultType"`
			Result     []QueryResult `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", result.Status)
	}

	return result.Data.Result, nil
}

// GetMetricLabels fetches all label names for a given metric
func (p *PrometheusClient) GetMetricLabels(ctx context.Context, metricName string) ([]string, error) {
	u := fmt.Sprintf("%s/api/v1/labels", p.baseURL)

	params := url.Values{}
	params.Add("match[]", metricName)

	req, err := http.NewRequestWithContext(ctx, "GET", u+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", result.Status)
	}

	return result.Data, nil
}

// MetricStats holds statistical information about a metric
type MetricStats struct {
	Min        float64
	Max        float64
	Avg        float64
	Rate       float64
	Count      int
	Labels     []string
	Error      error
	HasAvg     bool
	HasRate    bool
	IsSummary  bool
}

// GetMetricStats fetches statistics for a metric over the last hour
func (p *PrometheusClient) GetMetricStats(ctx context.Context, metricName string) MetricStats {
	stats := MetricStats{}

	// Get labels
	labels, err := p.GetMetricLabels(ctx, metricName)
	if err != nil {
		stats.Error = err
		return stats
	}
	stats.Labels = labels

	// Check if it's a summary (has quantile label)
	for _, label := range labels {
		if label == "quantile" {
			stats.IsSummary = true
			break
		}
	}

	// Get cardinality (number of time series)
	countResults, err := p.QueryInstant(ctx, fmt.Sprintf("count(%s)", metricName))
	if err == nil && len(countResults) > 0 && len(countResults[0].Value) > 1 {
		if val, ok := countResults[0].Value[1].(string); ok {
			fmt.Sscanf(val, "%d", &stats.Count)
		}
	}

	// Query for min/max
	queries := map[string]*float64{
		fmt.Sprintf("min(%s)", metricName): &stats.Min,
		fmt.Sprintf("max(%s)", metricName): &stats.Max,
	}

	// Only add avg/rate for non-summary metrics
	if !stats.IsSummary {
		queries[fmt.Sprintf("avg(%s)", metricName)] = &stats.Avg
		queries[fmt.Sprintf("rate(%s[1h])", metricName)] = &stats.Rate
		stats.HasAvg = true
		stats.HasRate = true
	}

	for query, target := range queries {
		results, err := p.QueryInstant(ctx, query)
		if err != nil {
			continue
		}
		if len(results) > 0 && len(results[0].Value) > 1 {
			if val, ok := results[0].Value[1].(string); ok {
				fmt.Sscanf(val, "%f", target)
			}
		}
	}

	return stats
}
