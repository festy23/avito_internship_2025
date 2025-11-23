//go:build load
// +build load

package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	baseURL        = "http://localhost:8080"
	targetRPS      = 5
	duration       = 30 * time.Second
	maxLatencyP99  = 300 * time.Millisecond
	minSuccessRate = 0.999 // 99.9%
	// RPS tolerance: allow ±10% deviation from target
	rpsTolerance = 0.1
)

type metrics struct {
	totalRequests   int
	successRequests int
	errorRequests   int
	latencies       []time.Duration
}

func TestLoad_CreatePR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Check if server is running and setup test data
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	healthResp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Server is not running at %s. Please start the server first with: docker-compose up\nError: %v", baseURL, err)
	}
	healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("Server health check failed with status %d", healthResp.StatusCode)
	}

	// Setup test data: create team and user if needed
	setupTestData(t, client)

	loadClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	metrics := &metrics{
		latencies: make([]time.Duration, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	interval := time.Second / time.Duration(targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			reqStart := time.Now()

			reqBody := map[string]string{
				"pull_request_id":   fmt.Sprintf("pr-load-%d", time.Now().UnixNano()),
				"pull_request_name": "Load Test PR",
				"author_id":         "u1",
			}

			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", baseURL+"/pullRequest/create", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := loadClient.Do(req)
			latency := time.Since(reqStart)
			metrics.latencies = append(metrics.latencies, latency)
			metrics.totalRequests++

			if err != nil {
				metrics.errorRequests++
				if metrics.errorRequests <= 3 {
					t.Logf("Request error: %v", err)
				}
				continue
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				metrics.successRequests++
			} else {
				metrics.errorRequests++
				if metrics.errorRequests <= 3 {
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Request failed: status=%d, body=%s", resp.StatusCode, string(body))
					resp.Body.Close()
				} else {
					resp.Body.Close()
				}
				continue
			}
			resp.Body.Close()
		}
	}

done:
	elapsed := time.Since(start)
	printMetrics(t, "CreatePR", metrics, elapsed)
	validateMetrics(t, metrics, elapsed)
}

func TestLoad_MergePR(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Check if server is running and setup test data
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	healthResp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Server is not running at %s. Please start the server first with: docker-compose up\nError: %v", baseURL, err)
	}
	healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("Server health check failed with status %d", healthResp.StatusCode)
	}

	// Setup test data: create team and user if needed
	setupTestData(t, client)

	// Setup: create a PR first
	prID := setupPR(t)

	loadClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	metrics := &metrics{
		latencies: make([]time.Duration, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	interval := time.Second / time.Duration(targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			reqStart := time.Now()

			reqBody := map[string]string{
				"pull_request_id": prID,
			}

			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", baseURL+"/pullRequest/merge", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := loadClient.Do(req)
			latency := time.Since(reqStart)
			metrics.latencies = append(metrics.latencies, latency)
			metrics.totalRequests++

			if err != nil {
				metrics.errorRequests++
				continue
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				metrics.successRequests++
			} else {
				metrics.errorRequests++
			}
			resp.Body.Close()
		}
	}

done:
	elapsed := time.Since(start)
	printMetrics(t, "MergePR", metrics, elapsed)
	validateMetrics(t, metrics, elapsed)
}

func TestLoad_GetStatistics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Check if server is running
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	healthResp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Server is not running at %s. Please start the server first with: docker-compose up\nError: %v", baseURL, err)
	}
	healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("Server health check failed with status %d", healthResp.StatusCode)
	}

	// Check if statistics endpoint exists
	statsResp, err := client.Get(baseURL + "/statistics/reviewers")
	if err != nil {
		t.Fatalf("Failed to reach statistics endpoint: %v", err)
	}
	statsResp.Body.Close()
	if statsResp.StatusCode == http.StatusNotFound {
		t.Fatalf("Statistics endpoint /statistics/reviewers not found (404). Check if endpoint is registered.")
	}

	client = &http.Client{
		Timeout: 10 * time.Second,
	}

	metrics := &metrics{
		latencies: make([]time.Duration, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	interval := time.Second / time.Duration(targetRPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			reqStart := time.Now()

			req, _ := http.NewRequest("GET", baseURL+"/statistics/reviewers", nil)

			resp, err := client.Do(req)
			latency := time.Since(reqStart)
			metrics.latencies = append(metrics.latencies, latency)
			metrics.totalRequests++

			if err != nil {
				metrics.errorRequests++
				if metrics.totalRequests <= 3 {
					t.Logf("Request error: %v", err)
				}
				continue
			}

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				metrics.successRequests++
			} else {
				metrics.errorRequests++
				if metrics.errorRequests <= 3 {
					body, _ := io.ReadAll(resp.Body)
					t.Logf("Request failed: status=%d, body=%s", resp.StatusCode, string(body))
					resp.Body.Close()
				} else {
					resp.Body.Close()
				}
				continue
			}
			resp.Body.Close()
		}
	}

done:
	elapsed := time.Since(start)
	printMetrics(t, "GetStatistics", metrics, elapsed)
	validateMetrics(t, metrics, elapsed)
}

func setupTestData(t *testing.T, client *http.Client) {
	// Create team "backend" with user "u1" if needed
	teamBody := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{
				"user_id":   "u1",
				"username":  "Alice",
				"is_active": true,
			},
			{
				"user_id":   "u2",
				"username":  "Bob",
				"is_active": true,
			},
			{
				"user_id":   "u3",
				"username":  "Charlie",
				"is_active": true,
			},
		},
	}

	body, _ := json.Marshal(teamBody)
	req, _ := http.NewRequest("POST", baseURL+"/team/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Warning: Failed to setup test data: %v", err)
		return
	}
	resp.Body.Close()
	// Ignore error if team already exists (400) - that's fine
}

func setupPR(t *testing.T) string {
	client := &http.Client{Timeout: 5 * time.Second}

	prID := fmt.Sprintf("pr-load-setup-%d", time.Now().UnixNano())
	reqBody := map[string]string{
		"pull_request_id":   prID,
		"pull_request_name": "Setup PR",
		"author_id":         "u1",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", baseURL+"/pullRequest/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return prID
}

func printMetrics(t *testing.T, testName string, m *metrics, elapsed time.Duration) {
	if len(m.latencies) == 0 {
		return
	}

	// Calculate percentiles
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sortDurations(sorted)

	p50 := sorted[len(sorted)*50/100]
	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]
	p999 := sorted[len(sorted)*999/1000]

	avgLatency := time.Duration(0)
	for _, lat := range m.latencies {
		avgLatency += lat
	}
	avgLatency /= time.Duration(len(m.latencies))

	successRate := float64(m.successRequests) / float64(m.totalRequests)
	actualRPS := float64(m.totalRequests) / elapsed.Seconds()

	t.Logf("\n=== Load Test Results: %s ===", testName)
	t.Logf("Duration: %v", elapsed)
	t.Logf("Total Requests: %d", m.totalRequests)
	t.Logf("Success Requests: %d", m.successRequests)
	t.Logf("Error Requests: %d", m.errorRequests)
	t.Logf("Success Rate: %.4f%%", successRate*100)
	t.Logf("Actual RPS: %.2f", actualRPS)
	t.Logf("Average Latency: %v", avgLatency)
	t.Logf("P50 Latency: %v", p50)
	t.Logf("P95 Latency: %v", p95)
	t.Logf("P99 Latency: %v", p99)
	t.Logf("P99.9 Latency: %v", p999)
}

func validateMetrics(t *testing.T, m *metrics, elapsed time.Duration) {
	if len(m.latencies) == 0 {
		return
	}

	successRate := float64(m.successRequests) / float64(m.totalRequests)

	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sortDurations(sorted)
	p99 := sorted[len(sorted)*99/100]

	// Calculate actual RPS
	actualRPS := float64(m.totalRequests) / elapsed.Seconds()
	minRPS := float64(targetRPS) * (1 - rpsTolerance)
	maxRPS := float64(targetRPS) * (1 + rpsTolerance)

	require.GreaterOrEqual(t, successRate, minSuccessRate,
		"Success rate %.4f%% is below required %.4f%%", successRate*100, minSuccessRate*100)

	require.LessOrEqual(t, p99, maxLatencyP99,
		"P99 latency %v exceeds maximum %v", p99, maxLatencyP99)

	require.GreaterOrEqual(t, actualRPS, minRPS,
		"Actual RPS %.2f is below minimum %.2f (target: %.2f)", actualRPS, minRPS, float64(targetRPS))

	require.LessOrEqual(t, actualRPS, maxRPS,
		"Actual RPS %.2f exceeds maximum %.2f (target: %.2f)", actualRPS, maxRPS, float64(targetRPS))
}

func sortDurations(durations []time.Duration) {
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
}
