package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hitechcloud-vietnam/har-skills"
)

// Basic visualization types
type Visualization struct {
	TotalRequests    int
	TotalSize        float64 // MB
	TotalDuration    float64 // ms
	Timeline         []TimelineEntry
	ContentTypeChart map[string]int
	DomainChart      map[string]int
	WaterfallData    []WaterfallEntry
}

// Timeline entry
type TimelineEntry struct {
	Timestamp time.Time
	EventType string
	URL       string
}

// Waterfall entry
type WaterfallEntry struct {
	URL      string
	Method   string
	Status   int
	Start    float64 // Relative time (ms)
	Duration float64 // Duration (ms)
	Size     float64 // Size (KB)
	Type     string  // Content type
	Blocked  float64 // Blocked time
	DNS      float64 // DNS lookup time
	Connect  float64 // Connection time
	Send     float64 // Send time
	Wait     float64 // Wait time
	Receive  float64 // Receive time
}

func main() {
	// Parse command-line arguments
	harPath := flag.String("file", "", "HAR file path")
	outputDir := flag.String("output", "har_viz", "Output directory")
	flag.Parse()

	// Validate the HAR file path
	if *harPath == "" {
		fmt.Println("Please provide a HAR file path using the -file flag.")
		flag.Usage()
		os.Exit(1)
	}

	// Create the output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Unable to create output directory: %v", err)
	}

	// Load the HAR file
	fmt.Printf("Analyzing HAR file: %s\n", *harPath)
	harFile, err := har.ParseFile(*harPath)
	if err != nil {
		log.Fatalf("Unable to parse HAR file: %v", err)
	}

	// Generate visualization data
	vizData := generateVisualization(harFile)

	// Write the visualization HTML
	htmlPath := fmt.Sprintf("%s/visualization.html", *outputDir)
	if err := generateHTML(vizData, htmlPath); err != nil {
		log.Fatalf("Failed to generate HTML: %v", err)
	}

	fmt.Printf("Visualization HTML generated: %s\n", htmlPath)
}

// Generate visualization data
func generateVisualization(harFile har.HARProvider) Visualization {
	viz := Visualization{
		ContentTypeChart: make(map[string]int),
		DomainChart:      make(map[string]int),
	}

	// Get all entries
	entries := harFile.GetEntries()
	if len(entries) == 0 {
		return viz
	}

	viz.TotalRequests = len(entries)

	// Find the earliest request time
	var firstRequestTime time.Time
	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()
		if firstRequestTime.IsZero() || entry.StartedDateTime.Before(firstRequestTime) {
			firstRequestTime = entry.StartedDateTime
		}
	}

	// Process each entry
	for _, entryProvider := range entries {
		entry := entryProvider.ToStandard()

		// Count entry information
		domain := extractDomain(entry.Request.URL)
		contentType := extractContentType(entry.Response.Content.MimeType)

		// Update total size
		viz.TotalSize += float64(entry.Response.Content.Size) / (1024 * 1024) // Convert to MB

		// Update total duration
		viz.TotalDuration += entry.Time

		// Update content-type statistics
		viz.ContentTypeChart[contentType]++

		// Update domain statistics
		viz.DomainChart[domain]++

		// Add to the timeline
		relativeStart := entry.StartedDateTime.Sub(firstRequestTime).Milliseconds()

		// Create a waterfall entry
		waterfall := WaterfallEntry{
			URL:      entry.Request.URL,
			Method:   entry.Request.Method,
			Status:   entry.Response.Status,
			Start:    float64(relativeStart),
			Duration: entry.Time,
			Size:     float64(entry.Response.Content.Size) / 1024, // KB
			Type:     contentType,
		}

		// Add timing information
		waterfall.Blocked = entry.Timings.Blocked
		waterfall.DNS = entry.Timings.DNS
		waterfall.Connect = entry.Timings.Connect
		waterfall.Send = entry.Timings.Send
		waterfall.Wait = entry.Timings.Wait
		waterfall.Receive = entry.Timings.Receive

		viz.WaterfallData = append(viz.WaterfallData, waterfall)
	}

	// Sort waterfall data by start time
	sort.Slice(viz.WaterfallData, func(i, j int) bool {
		return viz.WaterfallData[i].Start < viz.WaterfallData[j].Start
	})

	return viz
}

// Extract the domain from a URL.
func extractDomain(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.Split(url, "/")
	return parts[0]
}

// Extract the content type
func extractContentType(mimeType string) string {
	// Simplify the MIME type
	if strings.Contains(mimeType, "javascript") || strings.Contains(mimeType, "json") {
		return "JS"
	} else if strings.Contains(mimeType, "css") {
		return "CSS"
	} else if strings.Contains(mimeType, "html") {
		return "HTML"
	} else if strings.Contains(mimeType, "image") {
		return "Image"
	} else if strings.Contains(mimeType, "font") {
		return "Font"
	} else if strings.Contains(mimeType, "audio") || strings.Contains(mimeType, "video") {
		return "Media"
	} else if strings.Contains(mimeType, "text") {
		return "Text"
	}
	return "Other"
}

// Generate the HTML file
func generateHTML(viz Visualization, outputPath string) error {
	// Create the HTML file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the HTML header
	file.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>HAR File Visualization</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .section { margin-bottom: 30px; }
        .summary { display: flex; justify-content: space-between; flex-wrap: wrap; }
        .summary-item { 
            background: #f5f5f5; padding: 15px; border-radius: 5px; 
            flex: 1; min-width: 200px; margin: 5px; text-align: center;
        }
        .chart-container { display: flex; justify-content: space-between; flex-wrap: wrap; }
        .chart { flex: 1; min-width: 400px; height: 300px; margin: 10px; }
        .waterfall { 
            width: 100%; overflow-x: auto; margin-top: 20px;
            font-size: 12px; border-collapse: collapse;
        }
        .waterfall th, .waterfall td { padding: 5px; text-align: left; border-bottom: 1px solid #ddd; }
        .waterfall tr:hover { background-color: #f5f5f5; }
        .bar { height: 20px; position: relative; margin: 5px 0; }
        .bar-segment { position: absolute; height: 100%; }
        .blocked { background-color: #ccc; }
        .dns { background-color: #9C27B0; }
        .connect { background-color: #2196F3; }
        .send { background-color: #4CAF50; }
        .wait { background-color: #FF9800; }
        .receive { background-color: #F44336; }
        .legend { display: flex; margin: 10px 0; }
        .legend-item { display: flex; align-items: center; margin-right: 15px; }
        .legend-color { width: 15px; height: 15px; margin-right: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>HAR File Visualization</h1>
`)

	// Write summary information
	file.WriteString(fmt.Sprintf(`
        <div class="section">
            <h2>Summary</h2>
            <div class="summary">
                <div class="summary-item">
                    <h3>Total Requests</h3>
                    <div>%d</div>
                </div>
                <div class="summary-item">
                    <h3>Total Transfer Size</h3>
                    <div>%.2f MB</div>
                </div>
                <div class="summary-item">
                    <h3>Total Load Time</h3>
                    <div>%.2f s</div>
                </div>
            </div>
        </div>
`, viz.TotalRequests, viz.TotalSize, viz.TotalDuration/1000))

	// Write the charts section
	file.WriteString(`
        <div class="section">
            <h2>Charts</h2>
            <div class="chart-container">
                <div class="chart">
                    <canvas id="contentTypeChart"></canvas>
                </div>
                <div class="chart">
                    <canvas id="domainChart"></canvas>
                </div>
            </div>
        </div>
`)

	// Write the waterfall section
	file.WriteString(`
        <div class="section">
            <h2>Request Waterfall</h2>
            <div class="legend">
                <div class="legend-item"><div class="legend-color blocked"></div> Blocked</div>
                <div class="legend-item"><div class="legend-color dns"></div> DNS</div>
                <div class="legend-item"><div class="legend-color connect"></div> Connect</div>
                <div class="legend-item"><div class="legend-color send"></div> Send</div>
                <div class="legend-item"><div class="legend-color wait"></div> Wait</div>
                <div class="legend-item"><div class="legend-color receive"></div> Receive</div>
            </div>
            <table class="waterfall">
                <thead>
                    <tr>
                        <th>URL</th>
                        <th>Method</th>
                        <th>Status</th>
                        <th>Type</th>
                        <th>Size</th>
                        <th>Time</th>
                        <th>Timeline</th>
                    </tr>
                </thead>
                <tbody>
`)

	// Write waterfall rows
	maxTime := viz.WaterfallData[len(viz.WaterfallData)-1].Start + viz.WaterfallData[len(viz.WaterfallData)-1].Duration
	for _, entry := range viz.WaterfallData {
		// Calculate the start percentage
		startPercent := entry.Start / maxTime * 100

		// Calculate the duration-width percentage
		blockedWidth := entry.Blocked / entry.Duration * 100
		dnsWidth := entry.DNS / entry.Duration * 100
		connectWidth := entry.Connect / entry.Duration * 100
		sendWidth := entry.Send / entry.Duration * 100
		waitWidth := entry.Wait / entry.Duration * 100
		receiveWidth := entry.Receive / entry.Duration * 100

		// Simplify the displayed URL
		displayURL := entry.URL
		if len(displayURL) > 50 {
			displayURL = displayURL[:47] + "..."
		}

		file.WriteString(fmt.Sprintf(`
                <tr>
                    <td title="%s">%s</td>
                    <td>%s</td>
                    <td>%d</td>
                    <td>%s</td>
                    <td>%.1f KB</td>
                    <td>%.0f ms</td>
                    <td style="width: 400px;">
                        <div class="bar">
                            <div class="bar-segment blocked" style="left: %.1f%%; width: %.1f%%;"></div>
                            <div class="bar-segment dns" style="left: %.1f%%; width: %.1f%%;"></div>
                            <div class="bar-segment connect" style="left: %.1f%%; width: %.1f%%;"></div>
                            <div class="bar-segment send" style="left: %.1f%%; width: %.1f%%;"></div>
                            <div class="bar-segment wait" style="left: %.1f%%; width: %.1f%%;"></div>
                            <div class="bar-segment receive" style="left: %.1f%%; width: %.1f%%;"></div>
                        </div>
                    </td>
                </tr>
`,
			entry.URL, displayURL, entry.Method, entry.Status, entry.Type, entry.Size, entry.Duration,
			startPercent, blockedWidth,
			startPercent+blockedWidth, dnsWidth,
			startPercent+blockedWidth+dnsWidth, connectWidth,
			startPercent+blockedWidth+dnsWidth+connectWidth, sendWidth,
			startPercent+blockedWidth+dnsWidth+connectWidth+sendWidth, waitWidth,
			startPercent+blockedWidth+dnsWidth+connectWidth+sendWidth+waitWidth, receiveWidth,
		))
	}

	// Close the table and container
	file.WriteString(`
                </tbody>
            </table>
        </div>
    </div>
`)

	// Write the JavaScript chart code
	file.WriteString(`
    <script>
        // Content-type chart
        const contentTypeData = {
            labels: [`)

	// Write content-type labels
	var contentTypeLabels []string
	var contentTypeValues []int
	for label, value := range viz.ContentTypeChart {
		contentTypeLabels = append(contentTypeLabels, label)
		contentTypeValues = append(contentTypeValues, value)
	}
	for i, label := range contentTypeLabels {
		if i > 0 {
			file.WriteString(", ")
		}
		file.WriteString(fmt.Sprintf(`"%s"`, label))
	}

	file.WriteString(`],
            datasets: [{
                label: 'Request count',
                data: [`)

	// Write content-type values
	for i, value := range contentTypeValues {
		if i > 0 {
			file.WriteString(", ")
		}
		file.WriteString(fmt.Sprintf("%d", value))
	}

	file.WriteString(`],
                backgroundColor: [
                    '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF', '#FF9F40', '#C9CBCF'
                ]
            }]
        };

        // Domain chart
        const domainData = {
            labels: [`)

	// Write domain labels
	var domainLabels []string
	var domainValues []int
	for label, value := range viz.DomainChart {
		domainLabels = append(domainLabels, label)
		domainValues = append(domainValues, value)
	}
	for i, label := range domainLabels {
		if i > 0 {
			file.WriteString(", ")
		}
		file.WriteString(fmt.Sprintf(`"%s"`, label))
	}

	file.WriteString(`],
            datasets: [{
                label: 'Request count',
                data: [`)

	// Write domain values
	for i, value := range domainValues {
		if i > 0 {
			file.WriteString(", ")
		}
		file.WriteString(fmt.Sprintf("%d", value))
	}

	file.WriteString(`],
                backgroundColor: [
                    '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF', '#FF9F40', '#C9CBCF',
                    '#2E2EFE', '#088A08', '#FF0040', '#FF8000', '#01A9DB', '#D7DF01', '#6A0888'
                ]
            }]
        };

        // Render charts
        window.onload = function() {
            // Content-type chart
            new Chart(document.getElementById('contentTypeChart').getContext('2d'), {
                type: 'pie',
                data: contentTypeData,
                options: {
                    responsive: true,
                    plugins: {
                        title: {
                            display: true,
                            text: 'Distribution by Content Type'
                        }
                    }
                }
            });

            // Domain chart
            new Chart(document.getElementById('domainChart').getContext('2d'), {
                type: 'pie',
                data: domainData,
                options: {
                    responsive: true,
                    plugins: {
                        title: {
                            display: true,
                            text: 'Distribution by Domain'
                        }
                    }
                }
            });
        };
    </script>
</body>
</html>`)

	return nil
}
