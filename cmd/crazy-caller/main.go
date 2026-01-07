package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	URL         string
	Method      string
	RPS         int
	Duration    int
	Concurrency int
	Body        string
	PathParams  string // id=123,name=test
	QueryParams string
	Headers     string // Content-Type:application/json,Authorization:Bearer xxx
	Insecure    bool
	Timeout     int
}

type ResponseStats struct {
	TotalRequests int64
	SuccessCount  int64
	ErrorCount    int64
	TotalLatency  int64
	MinLatency    int64
	MaxLatency    int64
	StatusCodes   map[int]int64
	ErrorMessages map[string]int64
	mu            sync.RWMutex
}

func NewResponseStats() *ResponseStats {
	return &ResponseStats{
		StatusCodes:   make(map[int]int64),
		ErrorMessages: make(map[string]int64),
		MinLatency:    int64(^uint64(0) >> 1),
	}
}

type Result struct {
	StatusCode  int
	Latency     time.Duration
	Error       error
	BodyPreview string
	RequestURL  string
}

func main() {
	config := parseFlags()

	if config.URL == "" {
		log.Fatal("URL is required")
	}

	// 解析path parameters
	pathParams := parsePathParams(config.PathParams)

	// 准备HTTP客户端
	client := createHTTPClient(config)

	// 运行压力测试
	stats := runLoadTest(config, client, pathParams)

	// 打印结果
	printResults(stats, config.Duration)
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.URL, "url", "", "Base URL with variables like {id} or :id (required, must include protocol and port if needed)")
	flag.StringVar(&config.Method, "method", "GET", "HTTP method: GET, POST, PUT, DELETE, PATCH")
	flag.IntVar(&config.RPS, "rps", 10, "Requests per second")
	flag.IntVar(&config.Duration, "duration", 60, "Test duration in seconds")
	flag.IntVar(&config.Concurrency, "concurrency", 10, "Number of concurrent workers")
	flag.StringVar(&config.Body, "body", "", "Request body (JSON format)")
	flag.StringVar(&config.PathParams, "path-params", "", "Path parameters to replace in URL (format: id=123 or id=1-100)")
	flag.StringVar(&config.QueryParams, "query", "", "Query parameters (format: key1=value1&key2=value2)")
	flag.StringVar(&config.Headers, "headers", "", "HTTP headers (format: Content-Type:application/json,Authorization:Bearer xxx)")
	flag.BoolVar(&config.Insecure, "insecure", false, "Disable TLS certificate verification")
	flag.IntVar(&config.Timeout, "timeout", 10, "Request timeout in seconds")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "REST API Load Test Tool\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  1. Replace variables in URL:\n")
		fmt.Fprintf(os.Stderr, "     %s -url http://api.example.com/users/{id} -path-params id=1-100 -rps 50\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  2. Multiple path parameters:\n")
		fmt.Fprintf(os.Stderr, "     %s -url http://api.example.com/users/{userId}/orders/{orderId} \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "          -path-params userId=1-10,orderId=100-200 -rps 100\n\n")
		fmt.Fprintf(os.Stderr, "  3. POST with JSON body and headers:\n")
		fmt.Fprintf(os.Stderr, "     %s -url https://api.example.com/users -method POST \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "          -body '{\"name\":\"test\",\"age\":25}' \\\n")
		fmt.Fprintf(os.Stderr, "          -headers \"Content-Type:application/json,Authorization:Bearer token123\" \\\n")
		fmt.Fprintf(os.Stderr, "          -rps 50\n\n")
		fmt.Fprintf(os.Stderr, "  4. Complete example:\n")
		fmt.Fprintf(os.Stderr, "     %s -url https://api.example.com/products/{category}/{id} \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "          -path-params category=electronics|books|clothing,id=1-1000 \\\n")
		fmt.Fprintf(os.Stderr, "          -query \"page=1&limit=20\" \\\n")
		fmt.Fprintf(os.Stderr, "          -headers \"Accept:application/json,X-API-Key:your-key\" \\\n")
		fmt.Fprintf(os.Stderr, "          -rps 200 -concurrency 50 -duration 120\n")
	}

	flag.Parse()
	return config
}

func parsePathParams(paramStr string) map[string][]string {
	params := make(map[string][]string)
	if paramStr == "" {
		return params
	}

	pairs := strings.Split(paramStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// 检查是否是范围 (如 1-100)
			if strings.Contains(value, "-") {
				rangeParts := strings.Split(value, "-")
				if len(rangeParts) == 2 {
					start, err1 := strconv.Atoi(rangeParts[0])
					end, err2 := strconv.Atoi(rangeParts[1])
					if err1 == nil && err2 == nil && start <= end {
						var values []string
						for i := start; i <= end; i++ {
							values = append(values, strconv.Itoa(i))
						}
						params[key] = values
						continue
					}
				}
			}

			// 检查是否是列表 (如 value1|value2|value3)
			if strings.Contains(value, "|") {
				params[key] = strings.Split(value, "|")
				continue
			}

			// 单个值
			params[key] = []string{value}
		}
	}

	return params
}

func buildURL(baseURL string, requestIndex int, pathParams map[string][]string, queryParams string) string {
	urlStr := baseURL

	// 替换URL中的变量
	for key, values := range pathParams {
		if len(values) > 0 {
			// 轮询使用不同的值，使用模运算循环选择
			// 模运算确保 idx 始终在 [0, len(values)-1] 范围内，不会越界
			idx := requestIndex % len(values)
			value := values[idx]

			// 替换 {variable} 格式
			braceVar := fmt.Sprintf("{%s}", key)
			urlStr = strings.ReplaceAll(urlStr, braceVar, value)

			// 替换 :variable 格式
			colonVar := fmt.Sprintf(":%s", key)
			urlStr = strings.ReplaceAll(urlStr, colonVar, value)
		}
	}

	// 添加查询参数
	if queryParams != "" {
		if !strings.Contains(urlStr, "?") {
			urlStr += "?"
		} else {
			urlStr += "&"
		}
		urlStr += queryParams
	}

	return urlStr
}

func prepareBody(bodyStr string) ([]byte, error) {
	if bodyStr == "" {
		return nil, nil
	}

	// 验证是否为有效JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(bodyStr), &js); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %v", err)
	}

	return []byte(bodyStr), nil
}

func createHTTPClient(config *Config) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        config.Concurrency * 2,
		MaxIdleConnsPerHost: config.Concurrency,
		IdleConnTimeout:     90 * time.Second,
	}

	if config.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &http.Client{
		Transport: tr,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}
}

func createRequest(config *Config, requestIndex int, pathParams map[string][]string) (*http.Request, string, error) {
	// 构建URL
	finalURL := buildURL(config.URL, requestIndex, pathParams, config.QueryParams)

	var req *http.Request
	var err error

	if config.Body != "" {
		bodyBytes, err := prepareBody(config.Body)
		if err != nil {
			return nil, finalURL, err
		}
		req, err = http.NewRequest(config.Method, finalURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, finalURL, err
		}
	} else {
		req, err = http.NewRequest(config.Method, finalURL, nil)
		if err != nil {
			return nil, finalURL, err
		}
	}

	// 添加自定义headers
	if config.Headers != "" {
		headers := strings.Split(config.Headers, ",")
		for _, header := range headers {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	// 设置默认headers
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "crazy-caller/1.0")
	}

	// 如果没有设置Content-Type且body是JSON，自动设置
	if config.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 如果没有设置Accept，默认接受JSON
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	return req, finalURL, nil
}

func runLoadTest(config *Config, client *http.Client, pathParams map[string][]string) *ResponseStats {
	stats := NewResponseStats()

	// Distribute RPS accurately across workers
	baseRPS := config.RPS / config.Concurrency
	remainder := config.RPS % config.Concurrency

	stopChan := make(chan struct{})
	go func() {
		time.Sleep(time.Duration(config.Duration) * time.Second)
		close(stopChan)
	}()

	var wg sync.WaitGroup
	for i := 0; i < config.Concurrency; i++ {
		// First 'remainder' workers get baseRPS + 1, others get baseRPS
		workerRPS := baseRPS
		if i < remainder {
			workerRPS++
		}
		// Only start worker if it has RPS > 0
		if workerRPS > 0 {
			wg.Add(1)
			go worker(i, config, client, pathParams, workerRPS, stopChan, &wg, stats)
		}
	}

	wg.Wait()
	return stats
}

func worker(id int, config *Config, client *http.Client, pathParams map[string][]string, rps int, stopChan <-chan struct{}, wg *sync.WaitGroup, stats *ResponseStats) {
	defer wg.Done()

	// Each worker starts with a different offset to distribute parameter values across workers
	// This ensures different workers use different parameter values from the start
	requestIndex := id
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			req, finalURL, err := createRequest(config, requestIndex, pathParams)
			if err != nil {
				log.Printf("Worker %d: Failed to create request: %v", id, err)
				continue
			}

			sendRequest(client, req, finalURL, stats)
			requestIndex++

		case <-stopChan:
			return
		}
	}
}

func sendRequest(client *http.Client, req *http.Request, finalURL string, stats *ResponseStats) {
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)

	result := &Result{
		Latency:    latency,
		Error:      err,
		RequestURL: finalURL,
	}

	if err != nil {
		// Error already set above
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode

		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 100))
		if readErr != nil {
			result.BodyPreview = fmt.Sprintf("(error reading body: %v)", readErr)
		} else {
			result.BodyPreview = string(bodyBytes)
		}
	}

	updateStats(stats, result)
}

func updateStats(stats *ResponseStats, result *Result) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	atomic.AddInt64(&stats.TotalRequests, 1)

	if result.Error != nil {
		atomic.AddInt64(&stats.ErrorCount, 1)
		errorMsg := result.Error.Error()
		if len(errorMsg) > 50 {
			errorMsg = errorMsg[:50] + "..."
		}
		stats.ErrorMessages[errorMsg]++
		return
	}

	stats.StatusCodes[result.StatusCode]++

	success := false
	for _, code := range []int{200, 201, 202, 204} {
		if result.StatusCode == code {
			success = true
			break
		}
	}

	if success {
		atomic.AddInt64(&stats.SuccessCount, 1)
	} else {
		atomic.AddInt64(&stats.ErrorCount, 1)
		errorMsg := fmt.Sprintf("HTTP %d", result.StatusCode)
		stats.ErrorMessages[errorMsg]++
	}

	latencyMs := result.Latency.Milliseconds()
	atomic.AddInt64(&stats.TotalLatency, latencyMs)

	if latencyMs < stats.MinLatency {
		stats.MinLatency = latencyMs
	}

	if latencyMs > stats.MaxLatency {
		stats.MaxLatency = latencyMs
	}

	// 每1000个请求打印一次进度
	// Use atomic.LoadInt64 to safely read the atomic value
	total := atomic.LoadInt64(&stats.TotalRequests)
	if total > 0 && total%1000 == 0 {
		log.Printf("Progress: %d requests completed, last URL: %s", total, result.RequestURL)
	}
}

func printResults(stats *ResponseStats, duration int) {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("LOAD TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Test Duration:      %d seconds\n", duration)
	fmt.Printf("Total Requests:     %d\n", stats.TotalRequests)
	if duration > 0 {
		fmt.Printf("Requests/sec:       %.2f\n", float64(stats.TotalRequests)/float64(duration))
	} else {
		fmt.Printf("Requests/sec:       N/A (duration is 0)\n")
	}

	if stats.TotalRequests > 0 {
		successRate := float64(stats.SuccessCount) / float64(stats.TotalRequests) * 100
		fmt.Printf("Successful:         %d (%.2f%%)\n", stats.SuccessCount, successRate)
		fmt.Printf("Failed:             %d (%.2f%%)\n", stats.ErrorCount, 100-successRate)

		avgLatency := float64(stats.TotalLatency) / float64(stats.TotalRequests)
		fmt.Printf("\n=== Latency (ms) ===\n")
		fmt.Printf("Average:           %.2f ms\n", avgLatency)
		fmt.Printf("Min:               %d ms\n", stats.MinLatency)
		fmt.Printf("Max:               %d ms\n", stats.MaxLatency)
	}

	if len(stats.StatusCodes) > 0 {
		fmt.Println("\n=== Status Code Distribution ===")
		total := stats.TotalRequests
		for code, count := range stats.StatusCodes {
			percentage := float64(count) / float64(total) * 100
			fmt.Printf("  %d: %d (%.2f%%)\n", code, count, percentage)
		}
	}

	if len(stats.ErrorMessages) > 0 {
		fmt.Println("\n=== Error Summary ===")
		for err, count := range stats.ErrorMessages {
			fmt.Printf("  %s: %d\n", err, count)
		}
	}
	fmt.Println(strings.Repeat("=", 60))
}
