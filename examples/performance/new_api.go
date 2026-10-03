package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hitechcloud-vietnam/har-skills"
)

// Demonstrate the new functional-options API
func demonstrateNewApi() {
	exampleHarPath := "../../data/example.har"
	if _, err := os.Stat(exampleHarPath); os.IsNotExist(err) {
		log.Println("Example HAR file not found; skipping the new API demonstration")
		return
	}

	fmt.Println("\n=== New Functional-Options API Example ===")

	// Use standard parsing
	harData, err := har.ParseFile(exampleHarPath)
	if err != nil {
		log.Printf("Standard parsing failed: %v", err)
	} else {
		fmt.Printf("Standard parsing: loaded %d entries\n", len(harData.GetEntries()))
	}

	// Use memory-optimized parsing
	harData, err = har.ParseFile(exampleHarPath, har.WithMemoryOptimized())
	if err != nil {
		log.Printf("Memory-optimized parsing failed: %v", err)
	} else {
		fmt.Printf("Memory-optimized parsing: loaded %d entries\n", len(harData.GetEntries()))
	}

	// Use lazy parsing
	harData, err = har.ParseFile(exampleHarPath, har.WithLazyLoading())
	if err != nil {
		log.Printf("Lazy parsing failed: %v", err)
	} else {
		fmt.Printf("Lazy parsing: loaded %d entries\n", len(harData.GetEntries()))
	}

	// Use combined options
	harData, err = har.ParseFile(exampleHarPath, har.WithMemoryOptimized(), har.WithSkipValidation(), har.WithLenient())
	if err != nil {
		log.Printf("Combined-options parsing failed: %v", err)
	} else {
		fmt.Printf("Combined-options parsing: loaded %d entries\n", len(harData.GetEntries()))
	}

	// Use predefined option groups
	harData, err = har.ParseFile(exampleHarPath, har.OptMemoryEfficient...)
	if err != nil {
		log.Printf("Predefined-option parsing failed: %v", err)
	} else {
		fmt.Printf("Predefined-option parsing: loaded %d entries\n", len(harData.GetEntries()))
	}

	// Use streaming parsing
	iterator, err := har.NewStreamingParserFromFile(exampleHarPath)
	if err != nil {
		log.Printf("Failed to create the streaming parser: %v", err)
	} else {
		count := 0
		for iterator.Next() {
			count++
		}
		if err := iterator.Err(); err != nil {
			log.Printf("Error while streaming: %v", err)
		} else {
			fmt.Printf("Streaming parse: processed %d entries\n", count)
		}
	}

	// Use interfaces for generic processing
	harData, err = har.ParseFile(exampleHarPath)
	if err != nil {
		log.Printf("Interface parsing failed: %v", err)
	} else {
		fmt.Println("\nProcess HAR data through the interface:")
		processAnyHar(harData)
	}
}

// Generic processing function that accepts any type implementing HARProvider
func processAnyHar(har har.HARProvider) {
	fmt.Printf("HAR version: %s\n", har.GetVersion())
	fmt.Printf("Creator: %s %s\n", har.GetCreator().Name, har.GetCreator().Version)
	fmt.Printf("Entry count: %d\n", len(har.GetEntries()))

	// Process all entries
	if len(har.GetEntries()) > 0 {
		fmt.Println("\nFirst entry details:")
		entry := har.GetEntries()[0]
		request := entry.GetRequest()
		response := entry.GetResponse()

		fmt.Printf("  Request: %s %s\n", request.GetMethod(), request.GetURL())
		fmt.Printf("  Response: %d %s\n", response.GetStatus(), response.GetStatusText())
		fmt.Printf("  Content type: %s\n", response.GetContent().GetMimeType())
		fmt.Printf("  Content size: %d bytes\n", response.GetContent().GetSize())
	}
}
