package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hitechcloud-vietnam/har-skills"
)

func main() {
	fmt.Println("========= Enhanced HAR Error Handling Example =========")

	// Example 1: Use the enhanced parsing API to get detailed error information
	fmt.Println("\n[Example 1] Detailed Error Information")
	harFilePath := "../../data/www.google.com.har"
	harFile, harErr := har.ParseHarFileEnhanced(harFilePath)
	if harErr != nil {
		fmt.Println("Parsing failed; detailed error information:")
		printHarError(harErr, 0)
	} else {
		fmt.Printf("Successfully parsed the HAR file with %d request entries\n", len(harFile.Log.Entries))
	}

	// Example 2: Handle a missing file (filesystem error)
	fmt.Println("\n[Example 2] Handle a Filesystem Error")
	nonExistentFile := "non-existent.har"
	_, harErr = har.ParseHarFileEnhanced(nonExistentFile)
	if harErr != nil {
		fmt.Printf("Error code: %v\n", harErr.GetCode())
		fmt.Printf("Is filesystem error: %v\n", harErr.IsFileSystemError())
		fmt.Printf("Error message: %s\n", harErr.Error())

		if harErr.Metadata != nil {
			fmt.Println("Metadata:")
			for k, v := range harErr.Metadata {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}
	}

	// Example 3: Create JSON with invalid fields to test partial parsing
	fmt.Println("\n[Example 3] Partial Parsing — Handle Invalid Fields")
	invalidJSON := createInvalidJSON()
	tempFile := "temp_invalid.har"

	// Write a temporary file
	err := os.WriteFile(tempFile, []byte(invalidJSON), 0644)
	if err != nil {
		fmt.Println("Failed to create temporary file:", err)
		return
	}
	defer os.Remove(tempFile) // Clean up the temporary file.

	// Parse in lenient mode
	fmt.Println("Using lenient parsing mode::")
	harFile, err = har.ParseHarFileLenient(tempFile)
	if err != nil {
		if harErr, ok := err.(*har.HarError); ok {
			fmt.Println("The following warnings occurred during parsing, but a partial result was still returned::")
			printHarError(harErr, 0)

			// Print the successfully parsed portion
			if harFile != nil {
				fmt.Println("\nSuccessfully parsed portion::")
				fmt.Printf("  Version: %s\n", harFile.Log.Version)
				fmt.Printf("  Creator: %s %s\n", harFile.Log.Creator.Name, harFile.Log.Creator.Version)
				fmt.Printf("  Page count: %d\n", len(harFile.Log.Pages))
				fmt.Printf("  Entry count: %d\n", len(harFile.Log.Entries))
			}
		} else {
			fmt.Println("Parsing failed:", err)
		}
	} else {
		fmt.Println("Parsing completed successfully with no warnings")
	}

	// Example 4: Parse and collect warnings
	fmt.Println("\n[Example 4] Collect Warnings")
	result, err := har.ParseHarFileWithWarnings(tempFile)
	if err != nil {
		fmt.Println("Parsing failed completely:", err)
	} else {
		fmt.Printf("Parsing succeeded with %d warnings\n", len(result.Warnings))
		for i, warning := range result.Warnings {
			fmt.Printf("Warning %d: %s\n", i+1, warning.Error())
		}

		// Print the successfully parsed portion
		fmt.Println("\nSuccessfully parsed portion::")
		fmt.Printf("  Version: %s\n", result.Har.Log.Version)
		fmt.Printf("  Creator: %s %s\n", result.Har.Log.Creator.Name, result.Har.Log.Creator.Version)
		fmt.Printf("  Page count: %d\n", len(result.Har.Log.Pages))
		fmt.Printf("  Entry count: %d\n", len(result.Har.Log.Entries))
	}
}

// Print HAR error information, including nested partial errors
func printHarError(harErr *har.HarError, level int) {
	prefix := strings.Repeat("  ", level)
	fmt.Printf("%sError: %s\n", prefix, harErr.Message)

	if harErr.Field != "" {
		fmt.Printf("%sField: %s\n", prefix, harErr.Field)
	}

	if harErr.Metadata != nil && len(harErr.Metadata) > 0 {
		fmt.Printf("%sMetadata:\n", prefix)
		for k, v := range harErr.Metadata {
			fmt.Printf("%s  %s: %v\n", prefix, k, v)
		}
	}

	if harErr.HasPartialErrors() {
		fmt.Printf("%sContains %d partial errors:\n", prefix, len(harErr.GetPartialErrors()))
		for i, pe := range harErr.GetPartialErrors() {
			fmt.Printf("%sPartial error %d:\n", prefix, i+1)
			printHarError(pe, level+1)
		}
	}
}

// Create JSON containing invalid fields
func createInvalidJSON() string {
	return `{
		"log": {
			"version": "1.2",
			"creator": {
				"name": "Test case",
				"version": "1.0"
			},
			"pages": [
				{
					"startedDateTime": "invalid-date",
					"id": "page_1",
					"title": "Test page",
					"pageTimings": {
						"onContentLoad": "not a number",
						"onLoad": 500
					}
				}
			],
			"entries": [
				{
					"startedDateTime": "2023-01-01T00:00:00.000Z",
					"time": 100,
					"request": {
						"method": "GET",
						"url": "https://example.com",
						"httpVersion": "HTTP/1.1",
						"headers": [
							{
								"name": "Accept",
								"value": "text/html"
							}
						]
					},
					"response": {
						"status": "not a number",
						"statusText": "OK",
						"content": {
							"size": 1024,
							"mimeType": "text/html"
						}
					}
				},
				{
					"invalidField": "This triggers a parsing error",
					"request": {
						"method": "POST",
						"url": "https://example.com/api"
					}
				}
			]
		}
	}`
}
