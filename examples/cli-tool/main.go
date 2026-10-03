package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/hitechcloud-vietnam/har-skills"
)

const (
	VERSION = "0.1.0"
)

// Command-line arguments
type CommandArgs struct {
	HarFile   string
	Command   string
	Filter    string
	Format    string
	Limit     int
	SortField string
	SortOrder string
	Output    string
}

// Main function
func main() {
	// Parse command-line arguments
	args := parseArgs()

	// Validate the HAR file path
	if args.HarFile == "" {
		fmt.Println("Error: no HAR file path provided")
		printUsage()
		os.Exit(1)
	}

	// Load the HAR file
	harFile, err := har.ParseFile(args.HarFile, har.WithMemoryOptimized())
	if err != nil {
		log.Fatalf("Unable to parse HAR file: %v", err)
	}

	// Execute the command
	switch args.Command {
	case "info":
		showInfo(harFile)
	case "list":
		listEntries(harFile, args)
	case "find":
		findEntries(harFile, args)
	case "headers":
		showHeaders(harFile, args)
	case "timing":
		showTiming(harFile, args)
	case "extract":
		extractContent(harFile, args)
	default:
		fmt.Printf("Unknown command: %s\n", args.Command)
		printUsage()
		os.Exit(1)
	}
}

// Parse command-line arguments
func parseArgs() CommandArgs {
	// Define command-line arguments
	harFilePtr := flag.String("file", "", "HAR file path")
	commandPtr := flag.String("cmd", "info", "Command to run (info, list, find, headers, timing, extract)")
	filterPtr := flag.String("filter", "", "Filter criteria (URLregular expression、status code、type, etc.)")
	formatPtr := flag.String("format", "text", "Output format (text, json, csv)")
	limitPtr := flag.Int("limit", 10, "Maximum number of results")
	sortFieldPtr := flag.String("sort", "time", "Sort field (time, size, url, status)")
	sortOrderPtr := flag.String("order", "desc", "Sort order (asc, desc)")
	outputPtr := flag.String("output", "", "Output file path")

	// Custom usage instructions
	flag.Usage = printUsage

	// Parse arguments
	flag.Parse()

	return CommandArgs{
		HarFile:   *harFilePtr,
		Command:   *commandPtr,
		Filter:    *filterPtr,
		Format:    *formatPtr,
		Limit:     *limitPtr,
		SortField: *sortFieldPtr,
		SortOrder: *sortOrderPtr,
		Output:    *outputPtr,
	}
}

// Print usage instructions
func printUsage() {
	fmt.Printf("HAR CLI tool v%s - command-line analyzer for HTTP Archive files\n\n", VERSION)
	fmt.Println("Usage: har-cli -file <HAR file path> -cmd <command> [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  info      - Show basic HAR file information")
	fmt.Println("  list      - List requests in the HAR file")
	fmt.Println("  find      - Find requests matching the criteria")
	fmt.Println("  headers   - Show request or response headers")
	fmt.Println("  timing    - Show request timing analysis")
	fmt.Println("  extract   - Extract response content")
	fmt.Println("\noptions:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Println("  har-cli -file example.har -cmd info")
	fmt.Println("  har-cli -file example.har -cmd list -limit 20")
	fmt.Println("  har-cli -file example.har -cmd find -filter \"api/users\"")
	fmt.Println("  har-cli -file example.har -cmd timing -sort time -order desc")
}

// Show basic HAR file information
func showInfo(harFile har.HARProvider) {
	entries := harFile.GetEntries()
	pages := harFile.GetPages()
	creator := harFile.GetCreator()

	fmt.Println("=== HAR File Information ===")
	fmt.Printf("Version: %s\n", harFile.GetVersion())
	fmt.Printf("Creator: %s %s\n", creator.Name, creator.Version)
	fmt.Printf("Page count: %d\n", len(pages))
	fmt.Printf("Request count: %d\n", len(entries))

	// Calculate total request size and response time
	var totalSize int64
	var totalTime float64
	statusCodes := make(map[int]int)
	methods := make(map[string]int)
	contentTypes := make(map[string]int)
	domains := make(map[string]int)

	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()
		totalSize += int64(entry.Response.Content.Size)
		totalTime += entry.Time

		// Status code counts
		statusCodes[entry.Response.Status]++

		// Request method counts
		methods[entry.Request.Method]++

		// Content type counts
		contentType := entry.Response.Content.MimeType
		if contentType != "" {
			// Simplify content types
			contentType = strings.Split(contentType, ";")[0]
			contentTypes[contentType]++
		}

		// Domain counts
		domain := extractDomain(entry.Request.URL)
		domains[domain]++
	}

	fmt.Printf("\nTotal transfer size: %.2f MB\n", float64(totalSize)/(1024*1024))
	fmt.Printf("Total response time: %.2f s\n", totalTime/1000)

	if len(entries) > 0 {
		fmt.Printf("Average response size: %.2f KB\n", float64(totalSize)/float64(len(entries))/1024)
		fmt.Printf("Average response time: %.2f ms\n", totalTime/float64(len(entries)))
	}

	// Display the status code distribution
	fmt.Println("\nStatus Code Distribution:")
	for status, count := range statusCodes {
		fmt.Printf("  %d: %d (%.1f%%)\n", status, count, float64(count)/float64(len(entries))*100)
	}

	// Display request method distribution
	fmt.Println("\nRequest method distribution:")
	for method, count := range methods {
		fmt.Printf("  %s: %d (%.1f%%)\n", method, count, float64(count)/float64(len(entries))*100)
	}

	// Display the top domains
	fmt.Println("\nDomain distribution (Top 5):")
	domainList := sortMapByValue(domains)
	for i, item := range domainList {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s: %d (%.1f%%)\n", item.Key, item.Value, float64(item.Value)/float64(len(entries))*100)
	}

	// Display the top content types
	fmt.Println("\nContent Type Distribution (Top 5):")
	contentTypeList := sortMapByValue(contentTypes)
	for i, item := range contentTypeList {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s: %d (%.1f%%)\n", item.Key, item.Value, float64(item.Value)/float64(len(entries))*100)
	}
}

// List requests in the HAR file
func listEntries(harFile har.HARProvider, args CommandArgs) {
	entries := harFile.GetEntries()

	// Convert to standard format
	var standardEntries []har.Entries
	for _, entryProvider := range entries {
		standardEntries = append(standardEntries, entryProvider.ToStandard())
	}

	// Sort
	sortEntries(standardEntries, args.SortField, args.SortOrder)

	// Limit the number of results
	if args.Limit > 0 && args.Limit < len(standardEntries) {
		standardEntries = standardEntries[:args.Limit]
	}

	// Print results
	printEntries(standardEntries, args.Format, args.Output)
}

// Find requests by criteria
func findEntries(harFile har.HARProvider, args CommandArgs) {
	if args.Filter == "" {
		fmt.Println("Error: the find command requires the -filter flag")
		return
	}

	entries := harFile.GetEntries()
	var filtered []har.Entries

	// Check whether the filter is a status code
	statusCode, err := strconv.Atoi(args.Filter)
	isStatusFilter := (err == nil)

	// Create a regular expression
	var re *regexp.Regexp
	if !isStatusFilter {
		re, err = regexp.Compile(args.Filter)
		if err != nil {
			fmt.Printf("Warning: invalid regular expression '%s'; falling back to simple string matching\n", args.Filter)
			re = nil
		}
	}

	// Filter entries
	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()

		// Status code filter
		if isStatusFilter && entry.Response.Status == statusCode {
			filtered = append(filtered, entry)
			continue
		}

		// URL regex filter
		if !isStatusFilter {
			if re != nil && re.MatchString(entry.Request.URL) {
				filtered = append(filtered, entry)
			} else if !isStatusFilter && strings.Contains(entry.Request.URL, args.Filter) {
				filtered = append(filtered, entry)
			}
		}
	}

	fmt.Printf("Found %d matching requests\n", len(filtered))

	// Sort
	sortEntries(filtered, args.SortField, args.SortOrder)

	// Limit the number of results
	if args.Limit > 0 && args.Limit < len(filtered) {
		filtered = filtered[:args.Limit]
	}

	// Print results
	printEntries(filtered, args.Format, args.Output)
}

// Key-value pair used for sorting
type KeyValue struct {
	Key   string
	Value int
}

// Extract the domain from a URL
func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.Split(url, "/")
	return parts[0]
}

// Sort a map by value
func sortMapByValue(m map[string]int) []KeyValue {
	var ss []KeyValue
	for k, v := range m {
		ss = append(ss, KeyValue{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return ss
}

// Sort the entry list
func sortEntries(entries []har.Entries, field, order string) {
	sort.Slice(entries, func(i, j int) bool {
		var less bool
		switch strings.ToLower(field) {
		case "time":
			less = entries[i].Time < entries[j].Time
		case "size":
			less = entries[i].Response.Content.Size < entries[j].Response.Content.Size
		case "url":
			less = entries[i].Request.URL < entries[j].Request.URL
		case "status":
			less = entries[i].Response.Status < entries[j].Response.Status
		default:
			less = entries[i].Time < entries[j].Time
		}

		if strings.ToLower(order) == "desc" {
			return !less
		}
		return less
	})
}

// Print the entry list
func printEntries(entries []har.Entries, format, outputPath string) {
	if len(entries) == 0 {
		fmt.Println("No results")
		return
	}

	switch format {
	case "json":
		printEntriesJSON(entries, outputPath)
	case "csv":
		printEntriesCSV(entries, outputPath)
	default: // text
		printEntriesText(entries, outputPath)
	}
}

// Print entries (text format)
func printEntriesText(entries []har.Entries, outputPath string) {
	// Create the output stream
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.Create(outputPath)
		if err != nil {
			fmt.Printf("Unable to create output file: %v\n", err)
			output = os.Stdout
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	// Format as a table
	w := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Index\tMethod\tStatus\tSize(KB)\tTime(ms)\tURL")
	fmt.Fprintln(w, "----\t----\t----\t--------\t--------\t---")

	for i, entry := range entries {
		fmt.Fprintf(w, "%d\t%s\t%d\t%.1f\t%.1f\t%s\n",
			i+1,
			entry.Request.Method,
			entry.Response.Status,
			float64(entry.Response.Content.Size)/1024,
			entry.Time,
			entry.Request.URL,
		)
	}
	w.Flush()
}

// Print entries (JSON format)
func printEntriesJSON(entries []har.Entries, outputPath string) {
	// Create a simplified output structure
	type SimpleEntry struct {
		Method     string  `json:"method"`
		URL        string  `json:"url"`
		Status     int     `json:"status"`
		StatusText string  `json:"statusText"`
		MimeType   string  `json:"mimeType"`
		Size       int     `json:"size"`
		Time       float64 `json:"time"`
	}

	var simpleEntries []SimpleEntry
	for _, entry := range entries {
		simpleEntries = append(simpleEntries, SimpleEntry{
			Method:     entry.Request.Method,
			URL:        entry.Request.URL,
			Status:     entry.Response.Status,
			StatusText: entry.Response.StatusText,
			MimeType:   entry.Response.Content.MimeType,
			Size:       entry.Response.Content.Size,
			Time:       entry.Time,
		})
	}

	// Serialize as JSON
	jsonData, err := json.MarshalIndent(simpleEntries, "", "  ")
	if err != nil {
		fmt.Printf("JSON serialization failed: %v\n", err)
		return
	}

	// Write output
	if outputPath != "" {
		err := os.WriteFile(outputPath, jsonData, 0644)
		if err != nil {
			fmt.Printf("Failed to write file: %v\n", err)
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("Wrote %d records to %s\n", len(entries), outputPath)
		}
	} else {
		fmt.Println(string(jsonData))
	}
}

// Print entries (CSV format)
func printEntriesCSV(entries []har.Entries, outputPath string) {
	// Create the output stream
	var output *os.File
	var err error
	if outputPath != "" {
		output, err = os.Create(outputPath)
		if err != nil {
			fmt.Printf("Unable to create output file: %v\n", err)
			output = os.Stdout
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	// Write the CSV header
	fmt.Fprintln(output, "method,url,status,statusText,mimeType,size,time")

	// Write data rows
	for _, entry := range entries {
		// Handle commas in URLs to preserve valid CSV formatting
		url := strings.ReplaceAll(entry.Request.URL, ",", "%2C")
		statusText := strings.ReplaceAll(entry.Response.StatusText, ",", " ")
		mimeType := strings.ReplaceAll(entry.Response.Content.MimeType, ",", " ")

		fmt.Fprintf(output, "%s,%s,%d,%s,%s,%d,%.1f\n",
			entry.Request.Method,
			url,
			entry.Response.Status,
			statusText,
			mimeType,
			entry.Response.Content.Size,
			entry.Time,
		)
	}

	if outputPath != "" {
		fmt.Printf("Wrote %d records to %s\n", len(entries), outputPath)
	}
}

// Show request or response headers
func showHeaders(harFile har.HARProvider, args CommandArgs) {
	if args.Filter == "" {
		fmt.Println("Error: specify the -filter flag to match a URL")
		return
	}

	entries := harFile.GetEntries()
	found := false

	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()
		if strings.Contains(entry.Request.URL, args.Filter) {
			found = true
			fmt.Printf("=== %s %s ===\n", entry.Request.Method, entry.Request.URL)

			fmt.Println("\nRequest headers:")
			for _, header := range entry.Request.Headers {
				fmt.Printf("  %s: %s\n", header.Name, header.Value)
			}

			fmt.Println("\nResponse headers:")
			for _, header := range entry.Response.Headers {
				fmt.Printf("  %s: %s\n", header.Name, header.Value)
			}

			fmt.Println("\nResponse status:", entry.Response.Status, entry.Response.StatusText)
			fmt.Println("Content type:", entry.Response.Content.MimeType)
			fmt.Println("Content size:", entry.Response.Content.Size, "bytes")

			// If multiple requests match, process only the first
			if args.Limit <= 1 {
				break
			}
		}
	}

	if !found {
		fmt.Println("No matching requests found")
	}
}

// Show request timing analysis
func showTiming(harFile har.HARProvider, args CommandArgs) {
	entries := harFile.GetEntries()

	// Convert to standard format and preprocess
	var timingEntries []struct {
		URL     string
		Method  string
		Status  int
		Time    float64
		Blocked float64
		DNS     float64
		Connect float64
		Send    float64
		Wait    float64
		Receive float64
		SSL     float64
	}

	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()

		// If a filter is set, skip non-matching entries
		if args.Filter != "" && !strings.Contains(entry.Request.URL, args.Filter) {
			continue
		}

		timing := struct {
			URL     string
			Method  string
			Status  int
			Time    float64
			Blocked float64
			DNS     float64
			Connect float64
			Send    float64
			Wait    float64
			Receive float64
			SSL     float64
		}{
			URL:     entry.Request.URL,
			Method:  entry.Request.Method,
			Status:  entry.Response.Status,
			Time:    entry.Time,
			Blocked: entry.Timings.Blocked,
			DNS:     entry.Timings.DNS,
			Connect: entry.Timings.Connect,
			Send:    entry.Timings.Send,
			Wait:    entry.Timings.Wait,
			Receive: entry.Timings.Receive,
			SSL:     entry.Timings.Ssl,
		}

		timingEntries = append(timingEntries, timing)
	}

	// Sort
	sort.Slice(timingEntries, func(i, j int) bool {
		if args.SortField == "time" {
			return timingEntries[i].Time > timingEntries[j].Time
		}
		return timingEntries[i].Wait > timingEntries[j].Wait
	})

	// Limit the number of results
	if args.Limit > 0 && args.Limit < len(timingEntries) {
		timingEntries = timingEntries[:args.Limit]
	}

	// Print results
	fmt.Println("=== Request Timing Analysis ===")
	fmt.Println("(Time unit: ms)")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "URL\tTotal time\tBlocked\tDNS\tConnect\tSSL\tSend\tWait\tReceive")

	for _, t := range timingEntries {
		// Simplify the displayed URL
		url := t.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\n",
			url, t.Time, t.Blocked, t.DNS, t.Connect, t.SSL, t.Send, t.Wait, t.Receive)
	}
	w.Flush()
}

// Extract response content
func extractContent(harFile har.HARProvider, args CommandArgs) {
	if args.Filter == "" {
		fmt.Println("Error: specify the -filter flag to match a URL")
		return
	}

	if args.Output == "" {
		fmt.Println("Error: specify the -output flag to set the output file")
		return
	}

	entries := harFile.GetEntries()
	found := false

	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()
		if strings.Contains(entry.Request.URL, args.Filter) {
			found = true

			// Confirm the output file
			fmt.Printf("Found matching request: %s\n", entry.Request.URL)
			fmt.Printf("Content type: %s, size: %d bytes\n",
				entry.Response.Content.MimeType, entry.Response.Content.Size)

			// In production, content should be extracted from the HAR; this example uses simplified handling
			fmt.Println("Note: this version does not support extracting actual content from a HAR file")
			fmt.Printf("This feature will be added in a future version; output would be written to: %s\n", args.Output)

			// If multiple requests match, process only the first
			if args.Limit <= 1 {
				break
			}
		}
	}

	if !found {
		fmt.Println("No matching requests found")
	}
}
