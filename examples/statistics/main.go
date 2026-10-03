package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/har-skills"
)

// Statistics types
type Stats struct {
	TotalRequests       int
	TotalSize           int64
	TotalDuration       float64
	AvgRequestSize      float64
	AvgResponseTime     float64
	MedianResponseTime  float64
	MinResponseTime     float64
	MaxResponseTime     float64
	P95ResponseTime     float64
	StatusCodes         map[int]int
	ContentTypes        map[string]int
	DomainsCount        map[string]int
	SlowRequests        []RequestInfo
	LargeResponses      []RequestInfo
	TimeByContentType   map[string]float64
	SizeByContentType   map[string]int64
	RequestsPerSecond   map[int]int
	SuccessRate         float64
	CacheHitRate        float64
	PerformanceByDomain map[string]DomainPerformance
}

// Request information
type RequestInfo struct {
	URL          string
	Method       string
	StatusCode   int
	Size         int
	Duration     float64
	StartTime    time.Time
	ContentType  string
	Domain       string
	CacheStatus  string
	ResponseSize int
}

// Domain performance statistics
type DomainPerformance struct {
	Requests     int
	TotalTime    float64
	TotalSize    int64
	AvgTime      float64
	AvgSize      float64
	MaxTime      float64
	MinTime      float64
	SuccessCount int
	ErrorCount   int
}

func main() {
	// Parse command-line arguments
	harPath := flag.String("file", "", "HAR file path")
	outputFormat := flag.String("format", "text", "Output format: text or json")
	slowThreshold := flag.Float64("slow", 500, "Slow request threshold (ms)")
	largeThreshold := flag.Int("large", 1000000, "Large response threshold (bytes)")
	flag.Parse()

	// Validate the HAR file path
	if *harPath == "" {
		fmt.Println("Please provide a HAR file path using the -file flag.")
		flag.Usage()
		os.Exit(1)
	}

	// Load the HAR file using memory-optimized mode.
	fmt.Printf("Analyzing HAR file: %s\n", *harPath)
	harFile, err := har.ParseFile(*harPath, har.WithMemoryOptimized())
	if err != nil {
		log.Fatalf("Unable to parse HAR file: %v", err)
	}

	// Calculate statistics
	stats := calculateStats(harFile, *slowThreshold, *largeThreshold)

	// Print statistics
	switch *outputFormat {
	case "text":
		printTextStats(stats)
	case "json":
		printJSONStats(stats)
	default:
		fmt.Printf("Unsupported output format: %s\n", *outputFormat)
	}
}

// Calculate statistics
func calculateStats(harFile har.HARProvider, slowThreshold float64, largeThreshold int) Stats {
	stats := Stats{
		StatusCodes:         make(map[int]int),
		ContentTypes:        make(map[string]int),
		DomainsCount:        make(map[string]int),
		TimeByContentType:   make(map[string]float64),
		SizeByContentType:   make(map[string]int64),
		RequestsPerSecond:   make(map[int]int),
		PerformanceByDomain: make(map[string]DomainPerformance),
	}

	entries := harFile.GetEntries()
	if len(entries) == 0 {
		return stats
	}

	// Collect response times to calculate median and percentiles
	var responseTimes []float64
	var totalSuccessful, totalCached int

	// Earliest request time
	var firstRequestTime time.Time

	// Process each entry
	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()
		startTime := entry.StartedDateTime

		// Initialize the first request time
		if firstRequestTime.IsZero() || startTime.Before(firstRequestTime) {
			firstRequestTime = startTime
		}

		// Extract request information
		reqInfo := extractRequestInfo(entry)

		// Update basic count statistics
		stats.TotalRequests++
		stats.TotalSize += int64(reqInfo.Size)
		stats.TotalDuration += reqInfo.Duration
		stats.StatusCodes[reqInfo.StatusCode]++
		stats.ContentTypes[reqInfo.ContentType]++
		stats.DomainsCount[reqInfo.Domain]++

		// Count by content type
		baseContentType := getBaseContentType(reqInfo.ContentType)
		stats.TimeByContentType[baseContentType] += reqInfo.Duration
		stats.SizeByContentType[baseContentType] += int64(reqInfo.Size)

		// Count requests per second
		secondsSinceFirst := int(startTime.Sub(firstRequestTime).Seconds())
		stats.RequestsPerSecond[secondsSinceFirst]++

		// Collect response times
		responseTimes = append(responseTimes, reqInfo.Duration)

		// Slow requests and large responses
		if reqInfo.Duration > slowThreshold {
			stats.SlowRequests = append(stats.SlowRequests, reqInfo)
		}
		if reqInfo.Size > int(largeThreshold) {
			stats.LargeResponses = append(stats.LargeResponses, reqInfo)
		}

		// Count successful requests and cache hits
		if reqInfo.StatusCode >= 200 && reqInfo.StatusCode < 400 {
			totalSuccessful++
		}
		if reqInfo.CacheStatus == "hit" {
			totalCached++
		}

		// Count performance by domain
		domainPerf, exists := stats.PerformanceByDomain[reqInfo.Domain]
		if !exists {
			domainPerf = DomainPerformance{
				MinTime: math.MaxFloat64,
			}
		}
		domainPerf.Requests++
		domainPerf.TotalTime += reqInfo.Duration
		domainPerf.TotalSize += int64(reqInfo.Size)
		if reqInfo.Duration > domainPerf.MaxTime {
			domainPerf.MaxTime = reqInfo.Duration
		}
		if reqInfo.Duration < domainPerf.MinTime {
			domainPerf.MinTime = reqInfo.Duration
		}
		if reqInfo.StatusCode >= 200 && reqInfo.StatusCode < 400 {
			domainPerf.SuccessCount++
		} else {
			domainPerf.ErrorCount++
		}
		stats.PerformanceByDomain[reqInfo.Domain] = domainPerf
	}

	// Calculate averages
	stats.AvgRequestSize = float64(stats.TotalSize) / float64(stats.TotalRequests)
	stats.AvgResponseTime = stats.TotalDuration / float64(stats.TotalRequests)

	// Calculate success and cache-hit rates
	stats.SuccessRate = float64(totalSuccessful) / float64(stats.TotalRequests) * 100
	if totalCached > 0 {
		stats.CacheHitRate = float64(totalCached) / float64(stats.TotalRequests) * 100
	}

	// Calculate the median and percentiles
	if len(responseTimes) > 0 {
		sort.Float64s(responseTimes)
		stats.MinResponseTime = responseTimes[0]
		stats.MaxResponseTime = responseTimes[len(responseTimes)-1]
		stats.MedianResponseTime = percentile(responseTimes, 50)
		stats.P95ResponseTime = percentile(responseTimes, 95)
	}

	// Calculate average performance per domain
	for domain, perf := range stats.PerformanceByDomain {
		if perf.Requests > 0 {
			perf.AvgTime = perf.TotalTime / float64(perf.Requests)
			perf.AvgSize = float64(perf.TotalSize) / float64(perf.Requests)
			stats.PerformanceByDomain[domain] = perf
		}
	}

	return stats
}

// Extract request information
func extractRequestInfo(entry har.Entries) RequestInfo {
	info := RequestInfo{
		URL:         entry.Request.URL,
		Method:      entry.Request.Method,
		StatusCode:  entry.Response.Status,
		StartTime:   entry.StartedDateTime,
		Duration:    entry.Time,
		Size:        entry.Response.Content.Size,
		ContentType: entry.Response.Content.MimeType,
		CacheStatus: "miss", // Defaults to miss
	}

	// Extract domain
	info.Domain = extractDomain(entry.Request.URL)

	// Identify cache status — simplified; may need adjustment for actual use
	// Simplify handling because of the complexity of the actual HAR structure
	if entry.Cache.AfterRequest != nil {
		info.CacheStatus = "hit"
	}

	return info
}

// Extract the domain from a URL.
func extractDomain(url string) string {
	// Basic domain extraction; can be improved as needed
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.Split(url, "/")
	return parts[0]
}

// Get the base content type
func getBaseContentType(contentType string) string {
	parts := strings.Split(contentType, ";")
	baseType := strings.TrimSpace(parts[0])

	// Further categorize into major groups
	if strings.Contains(baseType, "javascript") || strings.Contains(baseType, "json") {
		return "javascript"
	} else if strings.Contains(baseType, "css") {
		return "css"
	} else if strings.Contains(baseType, "html") {
		return "html"
	} else if strings.Contains(baseType, "image") {
		return "image"
	} else if strings.Contains(baseType, "font") {
		return "font"
	} else if strings.Contains(baseType, "video") || strings.Contains(baseType, "audio") {
		return "media"
	}

	return baseType
}

// Calculate percentiles
func percentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	index := int(math.Ceil(float64(len(values))*percentile/100)) - 1
	if index < 0 {
		index = 0
	} else if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}

// Print statistics as text
func printTextStats(stats Stats) {
	fmt.Println("\n========== HAR File Statistics ==========")
	fmt.Printf("Total requests: %d\n", stats.TotalRequests)
	fmt.Printf("Total transfer size: %.2f MB\n", float64(stats.TotalSize)/(1024*1024))
	fmt.Printf("Total load time: %.2f s\n", stats.TotalDuration/1000)
	fmt.Printf("Average request size: %.2f KB\n", stats.AvgRequestSize/1024)
	fmt.Printf("Average response time: %.2f ms\n", stats.AvgResponseTime)
	fmt.Printf("Median response time: %.2f ms\n", stats.MedianResponseTime)
	fmt.Printf("Minimum response time: %.2f ms\n", stats.MinResponseTime)
	fmt.Printf("Maximum response time: %.2f ms\n", stats.MaxResponseTime)
	fmt.Printf("95th percentile response time: %.2f ms\n", stats.P95ResponseTime)
	fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate)
	fmt.Printf("Cache hit rate: %.2f%%\n", stats.CacheHitRate)

	fmt.Println("\n---------- Status Code Distribution ----------")
	for code, count := range stats.StatusCodes {
		fmt.Printf("%d: %d (%.2f%%)\n", code, count, float64(count)/float64(stats.TotalRequests)*100)
	}

	fmt.Println("\n---------- Content Type Distribution ----------")
	for contentType, count := range stats.ContentTypes {
		if count > 0 {
			fmt.Printf("%s: %d (%.2f%%)\n", contentType, count, float64(count)/float64(stats.TotalRequests)*100)
		}
	}

	fmt.Println("\n---------- Load Time by Content Type ----------")
	for contentType, time := range stats.TimeByContentType {
		if time > 0 {
			fmt.Printf("%s: %.2f ms (%.2f%%)\n", contentType, time, time/stats.TotalDuration*100)
		}
	}

	fmt.Println("\n---------- Transfer Size by Content Type ----------")
	for contentType, size := range stats.SizeByContentType {
		if size > 0 {
			fmt.Printf("%s: %.2f KB (%.2f%%)\n", contentType, float64(size)/1024, float64(size)/float64(stats.TotalSize)*100)
		}
	}

	fmt.Println("\n---------- Domain Performance ----------")
	// Sort by request count
	type DomainStat struct {
		Domain string
		Perf   DomainPerformance
	}
	var domainStats []DomainStat
	for domain, perf := range stats.PerformanceByDomain {
		domainStats = append(domainStats, DomainStat{domain, perf})
	}

	sort.Slice(domainStats, func(i, j int) bool {
		return domainStats[i].Perf.Requests > domainStats[j].Perf.Requests
	})

	for _, ds := range domainStats {
		fmt.Printf("%s:\n", ds.Domain)
		fmt.Printf("  Requests: %d (%.2f%%)\n", ds.Perf.Requests, float64(ds.Perf.Requests)/float64(stats.TotalRequests)*100)
		fmt.Printf("  Average response time: %.2f ms\n", ds.Perf.AvgTime)
		fmt.Printf("  Average size: %.2f KB\n", ds.Perf.AvgSize/1024)
		fmt.Printf("  Successful requests: %d, failed requests: %d\n", ds.Perf.SuccessCount, ds.Perf.ErrorCount)
	}

	fmt.Println("\n---------- Slowest Requests (Top 5) ----------")
	sort.Slice(stats.SlowRequests, func(i, j int) bool {
		return stats.SlowRequests[i].Duration > stats.SlowRequests[j].Duration
	})

	for i, req := range stats.SlowRequests {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s %s\n", i+1, req.Method, req.URL)
		fmt.Printf("   Response time: %.2f ms, status: %d, size: %.2f KB\n",
			req.Duration, req.StatusCode, float64(req.Size)/1024)
	}

	fmt.Println("\n---------- Largest Responses (Top 5) ----------")
	sort.Slice(stats.LargeResponses, func(i, j int) bool {
		return stats.LargeResponses[i].Size > stats.LargeResponses[j].Size
	})

	for i, req := range stats.LargeResponses {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s %s\n", i+1, req.Method, req.URL)
		fmt.Printf("   Size: %.2f KB, status: %d, response time: %.2f ms\n",
			float64(req.Size)/1024, req.StatusCode, req.Duration)
	}
}

// Print statistics as JSON
func printJSONStats(stats Stats) {
	// Implement JSON output here
	// For simplicity, this example only prints a text message
	fmt.Println("JSON output is under development...")
}
