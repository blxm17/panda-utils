# Testing Guide for crazy-caller

This document explains how to write and run tests in Go.

## Go Testing Basics

### Test File Naming
- Test files must end with `_test.go`
- Test files are in the same package as the code being tested
- Example: `main.go` → `main_test.go`

### Test Function Naming
- Test functions must start with `Test`
- Function signature: `func TestXxx(t *testing.T)`
- Example: `func TestParsePathParams(t *testing.T)`

### Running Tests

```bash
# Run all tests
go test ./cmd/crazy-caller

# Run with verbose output
go test -v ./cmd/crazy-caller

# Run a specific test
go test -v ./cmd/crazy-caller -run TestParsePathParams

# Run tests with coverage
go test -cover ./cmd/crazy-caller

# Generate coverage report
go test -coverprofile=coverage.out ./cmd/crazy-caller
go tool cover -html=coverage.out
```

## Test Organization

### Table-Driven Tests (Recommended Pattern)

Go uses table-driven tests as the standard pattern:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "test case 1",
            input:    "input1",
            expected: "output1",
        },
        {
            name:     "test case 2",
            input:    "input2",
            expected: "output2",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Function(tt.input)
            if result != tt.expected {
                t.Errorf("Function(%q) = %q, want %q", tt.input, result, tt.expected)
            }
        })
    }
}
```

### Subtests with t.Run()

- Use `t.Run()` to create subtests
- Each subtest runs independently
- Failed subtests don't stop other subtests
- Better test output and organization

### Test Helpers

Create helper functions for common test setup:

```go
func setupTestConfig() *Config {
    return &Config{
        URL:      "http://test.example.com",
        Method:   "GET",
        RPS:      10,
        Duration: 60,
    }
}
```

## Test Types

### Unit Tests
- Test individual functions in isolation
- Fast and focused
- Example: `TestParsePathParams`, `TestBuildURL`

### Integration Tests
- Test multiple components together
- May require external dependencies
- Use build tags: `//go:build integration`

### Benchmark Tests
- Measure performance
- Function name: `func BenchmarkXxx(b *testing.B)`
- Run with: `go test -bench=.`

## Best Practices

1. **Test Coverage**: Aim for >80% coverage on critical paths
2. **Test Names**: Use descriptive names that explain what is being tested
3. **Test Independence**: Each test should be independent and runnable in isolation
4. **Test Data**: Use clear, minimal test data
5. **Error Cases**: Always test error cases, not just happy paths
6. **Edge Cases**: Test boundary conditions (empty strings, nil, zero values)

## Example Test Structure

```
cmd/crazy-caller/
├── main.go              # Production code
├── main_test.go         # Main tests
├── parser_test.go       # Parser-specific tests
└── README_TEST.md       # This file
```

## Common Testing Patterns

### Testing Error Cases
```go
func TestFunction_ErrorCase(t *testing.T) {
    _, err := Function("invalid")
    if err == nil {
        t.Error("expected error, got nil")
    }
}
```

### Testing with Mocks
For HTTP clients, use `httptest` package:
```go
import "net/http/httptest"

func TestHTTPRequest(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    // Test with server.URL
}
```

### Testing Concurrent Code
Use `sync.WaitGroup` and channels to test concurrent behavior:
```go
func TestConcurrentFunction(t *testing.T) {
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Test concurrent access
        }()
    }
    wg.Wait()
}
```

