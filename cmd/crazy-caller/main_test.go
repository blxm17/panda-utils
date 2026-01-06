package main

import (
	"reflect"
	"testing"
)

// Test parsePathParams function
func TestParsePathParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string][]string{},
		},
		{
			name:  "single value",
			input: "id=123",
			expected: map[string][]string{
				"id": {"123"},
			},
		},
		{
			name:  "multiple single values",
			input: "id=123,name=test",
			expected: map[string][]string{
				"id":   {"123"},
				"name": {"test"},
			},
		},
		{
			name:  "range values",
			input: "id=1-5",
			expected: map[string][]string{
				"id": {"1", "2", "3", "4", "5"},
			},
		},
		{
			name:  "list values",
			input: "category=electronics|books|clothing",
			expected: map[string][]string{
				"category": {"electronics", "books", "clothing"},
			},
		},
		{
			name:  "mixed types",
			input: "id=1-3,category=electronics|books,name=test",
			expected: map[string][]string{
				"id":       {"1", "2", "3"},
				"category": {"electronics", "books"},
				"name":     {"test"},
			},
		},
		{
			name:  "with spaces",
			input: "id = 123 , name = test",
			expected: map[string][]string{
				"id":   {"123"},
				"name": {"test"},
			},
		},
		{
			name:  "invalid range (start > end)",
			input: "id=10-5",
			expected: map[string][]string{
				"id": {"10-5"}, // Falls back to single value
			},
		},
		{
			name:     "invalid format (no equals)",
			input:    "invalid",
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePathParams(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePathParams(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test buildURL function
func TestBuildURL(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		requestIndex int
		pathParams   map[string][]string
		queryParams  string
		expected     string
	}{
		{
			name:         "no params, no query",
			baseURL:      "http://api.example.com/users",
			requestIndex: 0,
			pathParams:   map[string][]string{},
			queryParams:  "",
			expected:     "http://api.example.com/users",
		},
		{
			name:         "single path param with brace format",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {"123"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/123",
		},
		{
			name:         "single path param with colon format",
			baseURL:      "http://api.example.com/users/:id",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {"123"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/123",
		},
		{
			name:         "multiple path params",
			baseURL:      "http://api.example.com/users/{userId}/orders/{orderId}",
			requestIndex: 0,
			pathParams: map[string][]string{
				"userId":  {"1"},
				"orderId": {"100"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/1/orders/100",
		},
		{
			name:         "path param with multiple values (cycling)",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {"1", "2", "3"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/1",
		},
		{
			name:         "path param cycling - index 1",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 1,
			pathParams: map[string][]string{
				"id": {"1", "2", "3"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/2",
		},
		{
			name:         "path param cycling - index 3 (wraps around)",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 3,
			pathParams: map[string][]string{
				"id": {"1", "2", "3"},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/1", // 3 % 3 = 0
		},
		{
			name:         "with query params",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {"123"},
			},
			queryParams: "page=1&limit=20",
			expected:    "http://api.example.com/users/123?page=1&limit=20",
		},
		{
			name:         "with query params (URL already has query)",
			baseURL:      "http://api.example.com/users/{id}?sort=asc",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {"123"},
			},
			queryParams: "page=1&limit=20",
			expected:    "http://api.example.com/users/123?sort=asc&page=1&limit=20",
		},
		{
			name:         "empty path params values",
			baseURL:      "http://api.example.com/users/{id}",
			requestIndex: 0,
			pathParams: map[string][]string{
				"id": {},
			},
			queryParams: "",
			expected:    "http://api.example.com/users/{id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildURL(tt.baseURL, tt.requestIndex, tt.pathParams, tt.queryParams)
			if result != tt.expected {
				t.Errorf("buildURL(%q, %d, %v, %q) = %q, want %q",
					tt.baseURL, tt.requestIndex, tt.pathParams, tt.queryParams, result, tt.expected)
			}
		})
	}
}

// Test prepareBody function
func TestPrepareBody(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantBytes []byte
		wantErr   bool
	}{
		{
			name:      "empty string",
			input:     "",
			wantBytes: nil,
			wantErr:   false,
		},
		{
			name:      "valid JSON object",
			input:     `{"name":"test","age":25}`,
			wantBytes: []byte(`{"name":"test","age":25}`),
			wantErr:   false,
		},
		{
			name:      "valid JSON array",
			input:     `[1,2,3]`,
			wantBytes: []byte(`[1,2,3]`),
			wantErr:   false,
		},
		{
			name:      "valid JSON string",
			input:     `"hello"`,
			wantBytes: []byte(`"hello"`),
			wantErr:   false,
		},
		{
			name:      "invalid JSON - missing quote",
			input:     `{"name":"test}`,
			wantBytes: nil,
			wantErr:   true,
		},
		{
			name:      "invalid JSON - trailing comma",
			input:     `{"name":"test",}`,
			wantBytes: nil,
			wantErr:   true,
		},
		{
			name:      "invalid JSON - unclosed bracket",
			input:     `{"name":"test"`,
			wantBytes: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prepareBody(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepareBody(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantBytes) {
				t.Errorf("prepareBody(%q) = %v, want %v", tt.input, got, tt.wantBytes)
			}
		})
	}
}

// Test NewResponseStats
func TestNewResponseStats(t *testing.T) {
	stats := NewResponseStats()

	if stats == nil {
		t.Fatal("NewResponseStats() returned nil")
	}

	if stats.StatusCodes == nil {
		t.Error("StatusCodes map is nil")
	}

	if stats.ErrorMessages == nil {
		t.Error("ErrorMessages map is nil")
	}

	if stats.MinLatency != int64(^uint64(0)>>1) {
		t.Errorf("MinLatency = %d, want %d", stats.MinLatency, int64(^uint64(0)>>1))
	}

	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}
}
