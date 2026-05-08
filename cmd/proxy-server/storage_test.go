package main

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewProxyStorage(t *testing.T) {
	// Test with a temporary file
	dbPath := "./test_storage.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }() // Ensure cleanup

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if storage == nil {
		t.Fatalf("Expected storage to be not nil")
	}
	defer storage.Close()

	// Check that the DB is set and not nil
	if storage.db == nil {
		t.Fatalf("Expected db to be not nil")
	}
}

func TestSaveRequestAndGetRequest(t *testing.T) {
	dbPath := "./test_storage2.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }()

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer storage.Close()

	// Create a sample RequestResponse
	rr := &RequestResponse{
		ID:             "test-id",
		Timestamp:      time.Now(),
		Method:         "GET",
		URL:            "http://example.com/path",
		Host:           "example.com",
		Path:           "/path",
		Query:          "",
		RequestHeaders: map[string]string{"User-Agent": "test"},
		RequestBody:    []byte("test body"),
		ResponseStatus: 200,
		ResponseHeaders: map[string]string{"Content-Type": "text/plain"},
		ResponseBody:   []byte("response body"),
		Duration:       100 * time.Millisecond,
		IsIntercepted:  false,
		Tags:           []string{"tag1", "tag2"},
	}

	// Save the request
	if err := storage.SaveRequest(rr); err != nil {
		t.Fatalf("Expected no error saving request, got %v", err)
	}

	// Retrieve the request
	rrGot, err := storage.GetRequest(rr.ID)
	if err != nil {
		t.Fatalf("Expected no error getting request, got %v", err)
	}
	if rrGot == nil {
		t.Fatalf("Expected request to be not nil")
	}

	// Compare the two (excluding timestamps which may differ slightly)
	rrGot.Timestamp = rr.Timestamp // Normalize timestamp for comparison
	if !reflect.DeepEqual(rr, rrGot) {
		t.Errorf("Request mismatch.\nWanted: %+v\nGot: %+v", rr, rrGot)
	}
}

func TestSaveRequest_Nil(t *testing.T) {
	dbPath := "./test_storage3.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }()

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer storage.Close()

	// Save a nil request should return an error
	err = storage.SaveRequest(nil)
	if err == nil {
		t.Fatalf("Expected error when saving nil request")
	}
}

func TestSearchRequests(t *testing.T) {
	dbPath := "./test_storage4.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }()

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer storage.Close()

	// Insert a few requests
	now := time.Now()
	rr1 := &RequestResponse{
		ID:        "1",
		Timestamp: now,
		Method:    "GET",
		URL:       "http://example.com/path1",
		Host:      "example.com",
		Path:      "/path1",
	}
	rr2 := &RequestResponse{
		ID:        "2",
		Timestamp: now.Add(-time.Hour),
		Method:    "POST",
		URL:       "http://example.org/path2",
		Host:      "example.org",
		Path:      "/path2",
	}
	rr3 := &RequestResponse{
		ID:        "3",
		Timestamp: now.Add(-2 * time.Hour),
		Method:    "GET",
		URL:       "http://example.com/path3",
		Host:      "example.com",
		Path:      "/path3",
		IsIntercepted: true,
	}

	for _, rr := range []*RequestResponse{rr1, rr2, rr3} {
		if err := storage.SaveRequest(rr); err != nil {
			t.Fatalf("Failed to save request: %v", err)
		}
	}

	// Test filter by host
	filters := RequestFilters{Host: "example.com"}
	results, err := storage.SearchRequests(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results for host example.com, got %d", len(results))
	}

	// Test filter by method
	filters = RequestFilters{Method: "POST"}
	results, err = storage.SearchRequests(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Method != "POST" {
		t.Fatalf("Expected 1 POST request, got %v", results)
	}

	// Test filter by intercepted
	filters = RequestFilters{InterceptedOnly: true}
	results, err = storage.SearchRequests(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].IsIntercepted {
		t.Fatalf("Expected 1 intercepted request, got %v", results)
	}

	// Test limit
	filters = RequestFilters{Limit: 1}
	results, err = storage.SearchRequests(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result with limit 1, got %d", len(results))
	}
}

func TestSaveFindingAndGetFindings(t *testing.T) {
	dbPath := "./test_storage5.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }()

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer storage.Close()

	// Create a sample SecurityFinding
	finding := &SecurityFinding{
		ID:          "finding-id",
		RequestID:   "request-id",
		Type:        "xss",
		Severity:    "high",
		Description: "Cross-site scripting",
		Evidence:    map[string]interface{}{"payload": "<script>alert(1)</script>"},
		CWE:         "CWE-79",
		CreatedAt:   time.Now(),
	}

	// Save the finding
	if err := storage.SaveFinding(finding); err != nil {
		t.Fatalf("Expected no error saving finding, got %v", err)
	}

	// Retrieve findings by type
	filters := FindingFilters{Type: "xss"}
	results, err := storage.GetFindings(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 finding, got %d", len(results))
	}
	// Normalize time for comparison
	finding.CreatedAt = results[0].CreatedAt
	if !reflect.DeepEqual(finding, results[0]) {
		t.Errorf("Finding mismatch.\nWanted: %+v\nGot: %+v", finding, results[0])
	}

	// Retrieve findings by severity
	filters = FindingFilters{Severity: "high"}
	results, err = storage.GetFindings(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 finding with severity high, got %d", len(results))
	}

	// Retrieve findings by minSeverity
	filters = FindingFilters{MinSeverity: "medium"}
	results, err = storage.GetFindings(filters)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 finding with minSeverity medium, got %d", len(results))
	}
}

func TestGetStats(t *testing.T) {
	dbPath := "./test_storage6.db"
	defer func() { _ = sql.Open("sqlite3", dbPath) }()

	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer storage.Close()

	// Initially, stats should be zero
	stats, err := storage.GetStats()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if stats.TotalRequests != 0 || stats.TotalFindings != 0 {
		t.Fatalf("Expected zero stats initially, got %+v", stats)
	}

	// Add a request and a finding
	rr := &RequestResponse{
		ID:        "req1",
		Timestamp: time.Now(),
		Method:    "GET",
		URL:       "http://example.com",
		Host:      "example.com",
	}
	if err := storage.SaveRequest(rr); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	finding := &SecurityFinding{
		ID:          "find1",
		RequestID:   "req1",
		Type:        "info",
		Severity:    "info",
		Description: "Info finding",
		Evidence:    map[string]interface{}{},
		CWE:         "",
		CreatedAt:   time.Now(),
	}
	if err := storage.SaveFinding(finding); err != nil {
		t.Fatalf("Failed to save finding: %v", err)
	}

	// Check stats again
	stats, err = storage.GetStats()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if stats.TotalRequests != 1 {
		t.Fatalf("Expected 1 request, got %d", stats.TotalRequests)
	}
	if stats.TotalFindings != 1 {
		t.Fatalf("Expected 1 finding, got %d", stats.TotalFindings)
	}
	if stats.FindingsBySeverity["info"] != 1 {
		t.Fatalf("Expected 1 info finding, got %d", stats.FindingsBySeverity["info"])
	}
}

func TestSeverityClause(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "1=1"},
		{"info", "LOWER(severity) IN ('info', 'low', 'medium', 'high', 'critical')"},
		{"low", "LOWER(severity) IN ('low', 'medium', 'high', 'critical')"},
		{"medium", "LOWER(severity) IN ('medium', 'high', 'critical')"},
		{"high", "LOWER(severity) IN ('high', 'critical')"},
		{"critical", "LOWER(severity) IN ('critical')"},
		{"INFO", "LOWER(severity) IN ('info', 'low', 'medium', 'high', 'critical')"},
		{"unknown", "1=1"}, // invalid severity defaults to 1=1
	}

	for _, tt := range tests {
		if got := severityClause(tt.input); got != tt.expected {
			t.Errorf("severityClause(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"info", "info"},
		{"informational", "info"},
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"critical", "critical"},
		{"INFO", "info"},
		{"INFORMATIONAL", "info"},
		{"LOW", "low"},
		{"MEDIUM", "medium"},
		{"HIGH", "high"},
		{"CRITICAL", "critical"},
		{"", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		if got := normalizeSeverity(tt.input); got != tt.expected {
			t.Errorf("normalizeSeverity(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}