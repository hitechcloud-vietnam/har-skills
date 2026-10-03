# Go-HAR Documentation

Go-HAR is a powerful and flexible Go library for parsing and processing HTTP Archive (HAR) files. This documentation will help you understand how to use Go-HAR and the features it provides.

## Feature Overview

Go-HAR provides the following major features:

* **Standard Parsing** - Parse HAR files and load them into memory
* **Memory-Optimized Parsing** - Reduce memory usage with optimized data structures
* **Lazy Loading** - Delay loading large content to reduce initial memory consumption
* **Streaming Parsing** - Process one entry at a time, suitable for extremely large HAR files
* **Enhanced Error Handling** - Provide detailed error and warning information
* **Conversion** - Convert between different parsing modes
* **Filtering and Searching** - Efficiently find and filter HAR data

## Installation

Use the `go get` command to add Go-HAR to your project:

```bash
go get github.com/hitechcloud-vietnam/har-skills
```

## Basic Usage

### Parsing a HAR File

The most basic usage is to parse a HAR file into an in-memory structure:

```go
package main

import (
    "fmt"
    "log"

    "github.com/hitechcloud-vietnam/har-skills"
)

func main() {
    // Parse the HAR file
    harData, err := har.ParseHarFile("example.har")
    if err != nil {
        log.Fatalf("Failed to parse HAR file: %v", err)
    }

    // Access HAR data
    fmt.Printf("HAR version: %s\n", harData.Log.Version)
    fmt.Printf("Number of entries: %d\n", len(harData.Log.Entries))

    // Iterate over all requests
    for i, entry := range harData.Log.Entries {
        fmt.Printf("Request #%d: %s %s\n",
            i+1,
            entry.Request.Method,
            entry.Request.URL,
        )
    }
}
```

### Using the Functional Options API (Recommended)

Go-HAR provides a flexible functional options API that allows you to customize parsing behavior:

```go
package main

import (
    "log"

    "github.com/hitechcloud-vietnam/har-skills"
)

func main() {
    // Parse the HAR file with options
    harData, err := har.ParseHarFile(
        "large.har",
        har.WithMemoryOptimized(), // Use memory-optimized mode
        har.WithValidation(false), // Skip validation for better performance
    )
    if err != nil {
        log.Fatalf("Failed to parse HAR file: %v", err)
    }

    // Process data through the interface
    ProcessEntries(harData)
}

// Using an interface allows any HAR implementation
func ProcessEntries(harData har.HARProvider) {
    for _, entry := range harData.GetEntries() {
        // Process each entry
        _ = entry.GetRequest().GetURL()
    }
}
```

## Advanced Usage

### Memory Optimization

For large HAR files, memory-optimized mode can significantly reduce memory usage:

```go
// Use memory-optimized mode
harData, err := har.ParseHarFile(
    "large.har",
    har.WithMemoryOptimized(),
)

if err != nil {
    log.Fatalf("Failed to parse HAR file: %v", err)
}

// The interface remains consistent
for _, entry := range harData.GetEntries() {
    fmt.Printf("URL: %s\n", entry.GetRequest().GetURL())
}
```

Memory-optimized mode uses the following techniques to reduce memory consumption:

* Use maps instead of arrays to store headers and query parameters
* Use pointers to represent optional fields
* Use enums instead of strings to store HTTP methods

### Lazy Loading

For HAR files containing large response bodies, lazy-loading mode can defer content loading:

```go
// Use lazy-loading mode
harData, err := har.ParseHarFile(
    "large_content.har",
    har.WithLazyLoading(),
)

if err != nil {
    log.Fatalf("Failed to parse HAR file: %v", err)
}

// Basic information is available immediately
for _, entry := range harData.GetEntries() {
    resp := entry.GetResponse()

    fmt.Printf(
        "Status code: %d, Content size: %d\n",
        resp.GetStatus(),
        resp.GetContent().GetSize(),
    )

    // Content is loaded only when needed
    if resp.GetStatus() == 200 {
        content := resp.GetContent()
        text := content.GetText() // Content is loaded at this point

        fmt.Printf("Content length: %d\n", len(text))
    }
}
```

### Streaming Parsing

For extremely large HAR files, streaming parsing allows entries to be processed one at a time:

```go
// Create a streaming parser
iterator, err := har.NewStreamingParserFromFile("huge.har")
if err != nil {
    log.Fatalf("Failed to create streaming parser: %v", err)
}

defer iterator.Close()

// Process entries one by one
for iterator.Next() {
    entry := iterator.Entry()

    fmt.Printf(
        "Processing request: %s\n",
        entry.GetRequest().GetURL(),
    )

    // The entry is released after processing
}

// Check whether an error occurred
if err := iterator.Error(); err != nil {
    log.Fatalf("Streaming parsing failed: %v", err)
}
```

### Enhanced Error Handling

For HAR files that may not fully comply with the standard, use enhanced error handling:

```go
// Use enhanced error handling
result, err := har.ParseHarFileWithWarnings("problematic.har")

if err != nil {
    log.Fatalf("Parsing failed completely: %v", err)
} else {
    // Parsing succeeded, but there may be warnings
    harData := result.HAR

    if len(result.Warnings) > 0 {
        fmt.Printf(
            "Parsing succeeded with %d warning(s):\n",
            len(result.Warnings),
        )

        for i, w := range result.Warnings {
            fmt.Printf(
                "Warning #%d: %s\n",
                i+1,
                w.Error(),
            )
        }
    }

    // Continue using harData
    fmt.Printf(
        "Successfully parsed %d entries\n",
        len(harData.GetEntries()),
    )
}
```

### Filtering and Searching

Go-HAR provides efficient filtering and searching capabilities:

```go
// Find all POST requests
postRequests := har.Filter(
    harData,
    func(entry har.EntryProvider) bool {
        return entry.GetRequest().GetMethod() == har.MethodPOST
    },
)

// Find all responses with status code 404
notFoundResponses := har.Filter(
    harData,
    func(entry har.EntryProvider) bool {
        return entry.GetResponse().GetStatus() == 404
    },
)

// Search for a specific URL pattern
apiCalls := har.Filter(
    harData,
    func(entry har.EntryProvider) bool {
        return strings.Contains(
            entry.GetRequest().GetURL(),
            "/api/v1/",
        )
    },
)

// Combine filtering conditions
slowApiCalls := har.Filter(
    harData,
    func(entry har.EntryProvider) bool {
        return strings.Contains(
            entry.GetRequest().GetURL(),
            "/api/",
        ) && entry.GetTime() > 1000 // Requests taking more than 1 second
    },
)
```

### Conversion

Convert between different parsing modes:

```go
// Convert from standard mode to memory-optimized mode
standardHar, _ := har.ParseHarFile("example.har")
optimizedHar := har.ToOptimized(standardHar)

// Convert from memory-optimized mode to standard mode
optimizedHar, _ := har.ParseHarFile(
    "example.har",
    har.WithMemoryOptimized(),
)

standardHar := optimizedHar.ToStandard()

// Convert any mode to standard mode through the interface
func ConvertToStandard(provider har.HARProvider) *har.Har {
    return provider.ToStandard()
}
```

## Utility Tools

Go-HAR also provides several utility tools to help you work with HAR files.

### Statistics Analysis

```go
// Analyze a HAR file and generate statistics
stats := har.AnalyzeStatistics(harData)

fmt.Printf("Total requests: %d\n", stats.TotalRequests)
fmt.Printf(
    "Average response time: %.2fms\n",
    stats.AverageResponseTime,
)

fmt.Printf(
    "Slowest request: %s (%.2fms)\n",
    stats.SlowestRequest.URL,
    stats.SlowestRequest.Time,
)

fmt.Printf(
    "Largest response: %s (%d bytes)\n",
    stats.LargestResponse.URL,
    stats.LargestResponse.Size,
)
```

### Visualization Tools

```go
// Create a waterfall chart
waterfall := har.CreateWaterfall(harData)

err := waterfall.SaveAsHTML("waterfall.html")
if err != nil {
    log.Fatalf("Failed to save waterfall chart: %v", err)
}

// Create a performance chart
perfChart := har.CreatePerformanceChart(harData)

err = perfChart.SaveAsHTML("performance.html")
if err != nil {
    log.Fatalf("Failed to save performance chart: %v", err)
}
```

### Command-Line Tool

Go-HAR also provides a command-line tool:

```bash
# Display basic HAR file information
go-har info example.har

# List all requests
go-har list example.har

# Find a specific request
go-har find example.har --url "/api"

# Display request headers
go-har headers example.har --url "/login"

# Analyze request timing
go-har timing example.har --sort-by time

# Extract content
go-har extract example.har --url "/api/data" --output data.json
```

## Reference

### Main Interfaces

Go-HAR is designed around interfaces. The primary interfaces include:

* `HARProvider` - Main interface for HAR files
* `EntryProvider` - Individual HTTP request/response entry
* `RequestProvider` - HTTP request
* `ResponseProvider` - HTTP response
* `ContentProvider` - Response content
* `HeadersProvider` - HTTP headers
* `TimingsProvider` - Request timing information

### Functional Options

Available functional options include:

* `WithMemoryOptimized()` - Use memory-optimized mode
* `WithLazyLoading()` - Use lazy-loading mode
* `WithValidation(bool)` - Enable or disable validation
* `WithWarnings()` - Collect warnings instead of returning errors
* `WithCacheEnabled(bool)` - Enable or disable content caching
* `WithMaxContentSize(int)` - Limit content size
* `WithIgnoreFields([]string)` - Ignore specific fields

## Conclusion

Go-HAR provides a flexible and powerful API for processing HAR files of various sizes. By selecting the appropriate parsing mode and leveraging the provided interfaces, you can efficiently process and analyze HTTP archive data regardless of its size or complexity.

For more detailed information, please refer to the source code documentation and the provided examples.
