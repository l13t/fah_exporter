package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Mock HTTP server for testing
func mockServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

func TestFetchTeamStats(t *testing.T) {
	server := mockServer(`{"id": 12345, "name": "Test Team", "founder": "Founder", "score": 1000, "wus": 10, "rank": 1}`, http.StatusOK)
	defer server.Close()

	client := &FoldingAtHomeClient{BaseURL: server.URL, TeamID: 12345}
	stats, err := client.FetchTeamStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.TeamName != "Test Team" {
		t.Errorf("Expected team name 'Test Team', got %s", stats.TeamName)
	}
	if stats.Score != 1000 {
		t.Errorf("Expected score 1000, got %d", stats.Score)
	}
}

func TestFetchUsersStats(t *testing.T) {
	server := mockServer(`[["name", "id", "rank", "score", "wus"], ["user1", 1, 1, 1000, 10]]`, http.StatusOK)
	defer server.Close()

	client := &FoldingAtHomeClient{BaseURL: server.URL, TeamID: 12345}
	stats, err := client.FetchUsersStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(*stats) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(*stats))
	}

	userStat := (*stats)[0]
	if userStat.User != "user1" {
		t.Errorf("Expected user 'user1', got %s", userStat.User)
	}
	if userStat.Score != 1000 {
		t.Errorf("Expected score 1000, got %d", userStat.Score)
	}
}

func TestFetchUserStats(t *testing.T) {
	server := mockServer(`{"name": "testuser", "score": 1000, "wus": 10, "rank": 1, "active_7_days": 5, "active_50_days": 20}`, http.StatusOK)
	defer server.Close()

	client := &FoldingAtHomeClient{BaseURL: server.URL, UserName: "testuser"}
	stats, err := client.FetchUserStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if stats.Name != "testuser" {
		t.Errorf("Expected user name 'testuser', got %s", stats.Name)
	}
	if stats.Score != 1000 {
		t.Errorf("Expected score 1000, got %d", stats.Score)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	client := &FoldingAtHomeClient{
		BaseURL:  "https://api.foldingathome.org",
		TeamID:   12345,
		UserName: "testuser",
	}
	exporter := NewExporter(client, "foldingathome")
	prometheus.MustRegister(exporter)

	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := promhttp.Handler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if len(rr.Body.String()) == 0 {
		t.Errorf("Handler returned empty body")
	}
}
