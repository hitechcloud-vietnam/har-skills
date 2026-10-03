package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/hitechcloud-vietnam/har-skills"
)

func main() {
	// Define command-line arguments.
	filePathPtr := flag.String("file", "", "HAR file path")
	modePtr := flag.String("mode", "all", "Performance test mode (standard, optimized, lazy, streaming, all)")
	flag.Parse()

	// If no file path was specified, check for the default example file
	filePath := *filePathPtr
	if filePath == "" {
		defaultPath := "../../data/example.har"
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			fmt.Println("Error: provide a HAR file path or place an example file at data/example.har")
			flag.Usage()
			return
		}
		filePath = defaultPath
		fmt.Printf("Using default HAR file: %s\n", filePath)
	}

	// Get the file size.
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Fatalf("Unable to get file information: %v", err)
	}
	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	fmt.Printf("File size: %.2f MB\n\n", fileSizeMB)

	mode := *modePtr
	switch mode {
	case "standard":
		benchmarkStandard(filePath)
	case "optimized":
		benchmarkOptimized(filePath)
	case "lazy":
		benchmarkLazy(filePath)
	case "streaming":
		benchmarkStreaming(filePath)
	case "all":
		benchmarkAll(filePath)
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		flag.Usage()
	}

	// Demonstrate the new API
	demonstrateNewApi()
}

// benchmarkAll runs all performance tests
func benchmarkAll(filePath string) {
	fmt.Println("=== Performance Comparison ===")
	fmt.Println("Testing memory use and performance for all parsing methods...")

	benchmarkStandard(filePath)
	benchmarkOptimized(filePath)
	benchmarkLazy(filePath)
	benchmarkStreaming(filePath)
}

// benchmarkStandard tests standard parsing performance
func benchmarkStandard(filePath string) {
	fmt.Println("\n=== Standard Parsing ===")
	start := time.Now()
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	memBefore := memStats.Alloc

	// Standard mode
	har, err := har.ParseFile(filePath)
	if err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}

	// Get memory usage
	runtime.ReadMemStats(&memStats)
	memAfter := memStats.Alloc
	memUsed := float64(memAfter-memBefore) / (1024 * 1024)
	elapsed := time.Since(start)

	// Calculate and display results
	entriesCount := len(har.GetEntries())
	fmt.Printf("Parse duration: %v\n", elapsed)
	fmt.Printf("Memory usage: %.2f MB\n", memUsed)
	fmt.Printf("Entry count: %d\n", entriesCount)
}

// benchmarkOptimized tests memory-optimized parsing performance
func benchmarkOptimized(filePath string) {
	fmt.Println("\n=== Memory-Optimized Parsing ===")
	start := time.Now()
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	memBefore := memStats.Alloc

	// Memory-optimized mode
	har, err := har.ParseFile(filePath, har.WithMemoryOptimized())
	if err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}

	// Get memory usage
	runtime.ReadMemStats(&memStats)
	memAfter := memStats.Alloc
	memUsed := float64(memAfter-memBefore) / (1024 * 1024)
	elapsed := time.Since(start)

	// Calculate and display results
	entriesCount := len(har.GetEntries())
	fmt.Printf("Parse duration: %v\n", elapsed)
	fmt.Printf("Memory usage: %.2f MB\n", memUsed)
	fmt.Printf("Entry count: %d\n", entriesCount)
}

// benchmarkLazy tests lazy parsing performance
func benchmarkLazy(filePath string) {
	fmt.Println("\n=== Lazy Parsing ===")
	start := time.Now()
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	memBefore := memStats.Alloc

	// Lazy mode
	har, err := har.ParseFile(filePath, har.WithLazyLoading())
	if err != nil {
		log.Fatalf("Parsing failed: %v", err)
	}

	// Get memory usage (initialization only)
	runtime.ReadMemStats(&memStats)
	memAfter := memStats.Alloc
	memUsed := float64(memAfter-memBefore) / (1024 * 1024)
	elapsed := time.Since(start)

	// Calculate and display results
	entriesCount := len(har.GetEntries())
	fmt.Printf("Parse duration: %v\n", elapsed)
	fmt.Printf("Initial memory usage: %.2f MB\n", memUsed)
	fmt.Printf("Entry count: %d\n", entriesCount)

	// Access the response content to trigger full loading
	if entriesCount > 0 {
		fmt.Println("\nAccessing the first entry content to trigger lazy loading...")
		runtime.GC()
		runtime.ReadMemStats(&memStats)
		memBefore = memStats.Alloc
		startAccess := time.Now()

		// Get the first entry content
		entry := har.GetEntries()[0]
		response := entry.GetResponse()
		content := response.GetContent()
		fmt.Printf("Content type: %s, size: %d bytes\n", content.GetMimeType(), content.GetSize())

		// Calculate access duration and additional memory
		runtime.ReadMemStats(&memStats)
		memAfter = memStats.Alloc
		additionalMem := float64(memAfter-memBefore) / (1024 * 1024)
		accessTime := time.Since(startAccess)
		fmt.Printf("Access duration: %v\n", accessTime)
		fmt.Printf("Additional memory usage: %.2f MB\n", additionalMem)
	}
}

// benchmarkStreaming tests streaming parsing performance
func benchmarkStreaming(filePath string) {
	fmt.Println("\n=== Streaming Parsing ===")
	start := time.Now()
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	memBefore := memStats.Alloc

	// Streaming mode
	iterator, err := har.NewStreamingParserFromFile(filePath)
	if err != nil {
		log.Fatalf("Failed to create the streaming parser: %v", err)
	}

	// Process all entries
	count := 0
	for iterator.Next() {
		count++
		// Fetch entries without processing them
		_ = iterator.Entry()
	}

	if err := iterator.Err(); err != nil {
		log.Fatalf("Error while streaming: %v", err)
	}

	// Get memory usage
	runtime.ReadMemStats(&memStats)
	memAfter := memStats.Alloc
	memUsed := float64(memAfter-memBefore) / (1024 * 1024)
	elapsed := time.Since(start)

	// Display the results.
	fmt.Printf("Parse duration: %v\n", elapsed)
	fmt.Printf("Memory usage: %.2f MB\n", memUsed)
	fmt.Printf("Entry count: %d\n", count)
}
