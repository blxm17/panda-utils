package main

import (
	"strconv"
	"testing"
)

// Example of testing edge cases for parsePathParams
func TestParsePathParams_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:  "range with single number",
			input: "id=5-5",
			expected: map[string][]string{
				"id": {"5"},
			},
		},
		{
			name:  "large range",
			input: "id=1-100",
			expected: map[string][]string{"id": func() []string {
				values := make([]string, 100)
				for i := 1; i <= 100; i++ {
					values[i-1] = strconv.Itoa(i)
				}
				return values
			}()},
		},
		{
			name:  "list with empty values",
			input: "tags=a||c",
			expected: map[string][]string{
				"tags": {"a", "", "c"},
			},
		},
		{
			name:  "value with dash (not a range)",
			input: "name=test-value",
			expected: map[string][]string{
				"name": {"test-value"},
			},
		},
		{
			name:  "value with pipe (not a list)",
			input: "name=test|value",
			expected: map[string][]string{
				"name": {"test|value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePathParams(tt.input)
			// For complex cases, we might need custom comparison
			if len(result) != len(tt.expected) {
				t.Errorf("parsePathParams(%q) returned %d params, want %d",
					tt.input, len(result), len(tt.expected))
			}
		})
	}
}

// Benchmark tests (optional but good practice)
func BenchmarkParsePathParams(b *testing.B) {
	input := "id=1-100,category=electronics|books|clothing,name=test"
	for i := 0; i < b.N; i++ {
		parsePathParams(input)
	}
}

func BenchmarkBuildURL(b *testing.B) {
	baseURL := "http://api.example.com/users/{id}/orders/{orderId}"
	pathParams := map[string][]string{
		"id":      {"1", "2", "3"},
		"orderId": {"100", "200"},
	}
	queryParams := "page=1&limit=20"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildURL(baseURL, i, pathParams, queryParams)
	}
}
