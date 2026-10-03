package main

import (
	"fmt"
	"os"

	"github.com/hitechcloud-vietnam/har-skills"
)

func main() {
	// Example 1: Parse an existing HAR file and use filtering features
	fmt.Println("====== Example 1: Parsing and Filtering ======")
	harFilePath := "../../data/www.google.com.har"
	harFile, err := har.ParseHarFile(harFilePath)
	if err != nil {
		fmt.Println("Failed to parse HAR file:", err)
		return
	}

	// Filter all POST requests
	postRequests := harFile.FindByMethod("POST")
	fmt.Printf("Found %d POST requests\n", postRequests.Count())

	// Filter all image requests
	imageRequests := harFile.Filter(har.FilterOptions{
		ContentType: "image/",
	})
	fmt.Printf("Found %d image requests\n", imageRequests.Count())

	// Filter all slow requests (over 500 ms)
	slowRequests := harFile.FindSlowRequests(500)
	fmt.Printf("Found %d slow requests (>500 ms)\n", slowRequests.Count())

	// Find failed requests
	errorRequests := harFile.FindErrors()
	fmt.Printf("Found %d failed requests\n", errorRequests.Count())

	// Example 2: Convert filtered results to CSV
	fmt.Println("\n====== Example 2: Conversion ======")
	if slowRequests.Count() > 0 {
		options := har.DefaultConvertOptions()
		options.IncludeTimings = true

		csvData, err := slowRequests.ToHar().Convert(har.FormatCSV, options)
		if err != nil {
			fmt.Println("Failed to convert to CSV:", err)
		} else {
			fmt.Println("Slow requests in CSV format:")
			fmt.Println(csvData)
		}

		// Convert to Markdown
		mdData, _ := slowRequests.ToHar().Convert(har.FormatMarkdown, options)
		fmt.Println("Slow requests in Markdown format (partial output):")
		lines := splitLines(mdData)
		if len(lines) > 5 {
			fmt.Println(lines[0])
			fmt.Println(lines[1])
			fmt.Println(lines[2])
			fmt.Println("... [more lines omitted] ...")
		} else {
			fmt.Println(mdData)
		}
	}

	// Example 3: Create a new HAR file
	fmt.Println("\n====== Example 3: Create a HAR file ======")
	newHar := har.NewHar()
	newHar.SetCreator("go-har-example", "1.0")

	// Add a page
	page := newHar.AddPage("page1", "Example Page")
	page.SetPageTimings(100, 300)

	// Add request/response entries
	entry := newHar.AddEntry("GET", "https://example.com/api/data", "HTTP/1.1", "page1")
	entry.AddRequestHeader("Accept", "application/json")
	entry.AddRequestHeader("User-Agent", "go-har/1.0")

	entry.SetResponseStatus(200, "OK")
	entry.AddResponseHeader("Content-Type", "application/json")
	entry.SetResponseContent(1024, "application/json")
	entry.SetTimings(10, 20, 30, 5, 50, 30, 25)

	// Save to a file
	newHarPath := "./generated.har"
	err = newHar.SaveToFile(newHarPath, true)
	if err != nil {
		fmt.Println("Failed to save HAR file:", err)
	} else {
		fmt.Printf("Successfully created and saved HAR file to %s\n", newHarPath)
	}

	// Read the saved file and verify it
	generatedHar, err := har.ParseHarFile(newHarPath)
	if err != nil {
		fmt.Println("Failed to parse the generated HAR file:", err)
	} else {
		fmt.Println("Successfully read the generated HAR file")
		fmt.Printf("Pages: %d, request entries: %d\n",
			len(generatedHar.Log.Pages), len(generatedHar.Log.Entries))
	}

	// Clean up the test file
	os.Remove(newHarPath)
}

// Helper function: split a string into lines
func splitLines(s string) []string {
	var lines []string
	var line string
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
