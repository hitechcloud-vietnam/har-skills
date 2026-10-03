package cmd

import (
	"fmt"
	"strings"
	"time"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// waterfallCmd generates a waterfall timeline analysis.
var waterfallCmd = &cobra.Command{
	Use:   "waterfall",
	Short: "Generate a waterfall timeline analysis",
	Long: `Generate a request waterfall timeline for a HAR file, showing request
timing relationships and detailed timing phases.

Supports critical path analysis, concurrency analysis, SLA compliance checks, and page timing metrics.

Examples:
  har -f capture.har waterfall
  har -f capture.har waterfall --critical-path
  har -f capture.har waterfall --concurrency
  har -f capture.har waterfall --sla "Home:/:2000" "API:/api:500"
  har -f capture.har waterfall --page-timings`,
	RunE: runWaterfall,
}

func init() {
	rootCmd.AddCommand(waterfallCmd)

	waterfallCmd.Flags().Bool("critical-path", false, "Show the critical path (longest request dependency chain)")
	waterfallCmd.Flags().Bool("concurrency", false, "Show the concurrency timeline")
	waterfallCmd.Flags().StringSlice("sla", nil, "SLA rules (format: name:urlPattern:maxDurationMs)")
	waterfallCmd.Flags().Bool("page-timings", false, "Show page timing metrics")
}

func runWaterfall(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	showCriticalPath, _ := cmd.Flags().GetBool("critical-path")
	showConcurrency, _ := cmd.Flags().GetBool("concurrency")
	slaRules, _ := cmd.Flags().GetStringSlice("sla")
	showPageTimings, _ := cmd.Flags().GetBool("page-timings")

	// Select the output based on the flags.
	if showCriticalPath {
		path := h.CriticalPath()
		return internal.WriteOutput(cmd, path, func() string {
			return formatCriticalPath(path)
		}, nil)
	}

	if showConcurrency {
		timeline := h.ConcurrencyTimeline()
		return internal.WriteOutput(cmd, timeline, func() string {
			return formatConcurrencyTimeline(timeline)
		}, nil)
	}

	if len(slaRules) > 0 {
		rules, err := parseSLARules(slaRules)
		if err != nil {
			return fmt.Errorf("failed to parse SLA rules: %w", err)
		}
		results := h.SLACheck(rules)
		return internal.WriteOutput(cmd, results, func() string {
			return formatSLAResults(results)
		}, nil)
	}

	if showPageTimings {
		metrics := h.PageTimingMetrics()
		return internal.WriteOutput(cmd, metrics, func() string {
			return formatPageTimings(metrics)
		}, nil)
	}

	// Show the waterfall by default.
	entries := h.Waterfall()
	return internal.WriteOutput(cmd, entries, func() string {
		return formatWaterfall(entries)
	}, nil)
}

// parseSLARules parses SLA rule strings.
func parseSLARules(rules []string) ([]har.SLARule, error) {
	var result []har.SLARule
	for _, r := range rules {
		parts := strings.SplitN(r, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected SLA rule format name:urlPattern:maxDurationMs, got: '%s'", r)
		}
		maxMs, err := time.ParseDuration(parts[2])
		if err != nil {
			// Try parsing the value as milliseconds.
			var msInt int
			n, _ := fmt.Sscanf(parts[2], "%d", &msInt)
			if n != 1 {
				return nil, fmt.Errorf("unable to parse maximum SLA duration '%s': %w", parts[2], err)
			}
			maxMs = time.Duration(msInt) * time.Millisecond
		}
		result = append(result, har.SLARule{
			Name:       strings.TrimSpace(parts[0]),
			URLPattern: strings.TrimSpace(parts[1]),
			MaxTime:    maxMs,
		})
	}
	return result, nil
}

// formatWaterfall formats the waterfall as ASCII text.
func formatWaterfall(entries []har.WaterfallEntry) string {
	if len(entries) == 0 {
		return "No waterfall data.\n"
	}

	var sb strings.Builder
	sb.WriteString("Request Waterfall\n")
	sb.WriteString("==========\n\n")

	// Calculate the total time range.
	maxEnd := time.Duration(0)
	for _, e := range entries {
		if e.EndTime > maxEnd {
			maxEnd = e.EndTime
		}
	}

	// Show each entry on one line, using ASCII characters to represent its time range.
	barWidth := 50 // ASCII bar width.
	scale := float64(barWidth) / float64(maxEnd.Milliseconds())

	for _, e := range entries {
		startPos := int(float64(e.StartTime.Milliseconds()) * scale)
		endPos := int(float64(e.EndTime.Milliseconds()) * scale)
		barLen := endPos - startPos
		if barLen < 1 {
			barLen = 1
		}

		urlDisplay := e.URL
		if len(urlDisplay) > 40 {
			urlDisplay = urlDisplay[:37] + "..."
		}

		// Build the timing bar.
		bar := strings.Repeat(" ", startPos) + strings.Repeat("#", barLen)

		sb.WriteString(fmt.Sprintf("#%2d %s %-6dms [%s]\n",
			e.Index, urlDisplay, e.Duration.Milliseconds(), bar))
	}

	sb.WriteString(fmt.Sprintf("\nTotal duration: %.1fms\n", float64(maxEnd.Milliseconds())))

	return sb.String()
}

// formatCriticalPath formats the critical path as text.
func formatCriticalPath(path []har.WaterfallEntry) string {
	if len(path) == 0 {
		return "No critical path data.\n"
	}

	var sb strings.Builder
	sb.WriteString("Critical Path Analysis\n")
	sb.WriteString("============\n\n")

	totalDuration := time.Duration(0)
	for i, e := range path {
		totalDuration += e.Duration
		urlDisplay := e.URL
		if len(urlDisplay) > 60 {
			urlDisplay = urlDisplay[:57] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d. #%d %s %s (%dms)\n",
			i+1, e.Index, e.Method, urlDisplay, e.Duration.Milliseconds()))
	}

	sb.WriteString(fmt.Sprintf("\nTotal critical path duration: %dms\n", totalDuration.Milliseconds()))
	return sb.String()
}

// formatConcurrencyTimeline formats the concurrency timeline as text.
func formatConcurrencyTimeline(timeline []har.ConcurrencyPoint) string {
	if len(timeline) == 0 {
		return "No concurrency data.\n"
	}

	var sb strings.Builder
	sb.WriteString("Concurrency Timeline\n")
	sb.WriteString("============\n\n")

	sb.WriteString(fmt.Sprintf("%-12s %-6s %s\n", "Time", "Concurrent", "Active Entries"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, p := range timeline {
		indices := fmt.Sprintf("%v", p.ActiveEntries)
		if len(indices) > 40 {
			indices = indices[:37] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-12s %-6d %s\n",
			fmt.Sprintf("%.0fms", float64(p.Time.Milliseconds())), p.ActiveCount, indices))
	}

	return sb.String()
}

// formatSLAResults formats SLA check results as text.
func formatSLAResults(results []har.SLAResult) string {
	if len(results) == 0 {
		return "No SLA check results.\n"
	}

	var sb strings.Builder
	sb.WriteString("SLA Compliance Check\n")
	sb.WriteString("============\n\n")

	sb.WriteString(fmt.Sprintf("%-15s %-6s %-10s %-10s %s\n",
		"Rule", "Passed", "Actual", "Maximum", "Overrun"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		overshoot := ""
		if r.Overshoot > 0 {
			overshoot = fmt.Sprintf("+%dms", r.Overshoot.Milliseconds())
		}
		sb.WriteString(fmt.Sprintf("%-15s %-6s %-10s %-10s %s\n",
			r.Rule.Name, status,
			fmt.Sprintf("%dms", r.Actual.Milliseconds()),
			fmt.Sprintf("%dms", r.Rule.MaxTime.Milliseconds()),
			overshoot))
	}

	return sb.String()
}

// formatPageTimings formats page timing metrics as text.
func formatPageTimings(metrics *har.PageTimingMetrics) string {
	if metrics == nil {
		return "No page timing data.\n"
	}

	var sb strings.Builder
	sb.WriteString("Page Timing Metrics\n")
	sb.WriteString("============\n\n")

	sb.WriteString(fmt.Sprintf("TTFB:             %dms\n", metrics.TTFB.Milliseconds()))
	sb.WriteString(fmt.Sprintf("DOMContentLoaded: %dms\n", metrics.DOMContentLoaded.Milliseconds()))
	sb.WriteString(fmt.Sprintf("OnLoad:           %dms\n", metrics.OnLoad.Milliseconds()))
	sb.WriteString(fmt.Sprintf("Total time:        %dms\n", metrics.TotalTime.Milliseconds()))
	sb.WriteString(fmt.Sprintf("DNS lookup:        %dms\n", metrics.DNSLookup.Milliseconds()))
	sb.WriteString(fmt.Sprintf("Connect time:      %dms\n", metrics.ConnectTime.Milliseconds()))
	sb.WriteString(fmt.Sprintf("SSL time:           %dms\n", metrics.SSLTime.Milliseconds()))

	return sb.String()
}
