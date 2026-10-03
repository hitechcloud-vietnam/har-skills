package cmd

import (
	"fmt"
	"sort"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// infoCmd displays a HAR file summary and statistics.
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show a HAR file summary and statistics",
	Long: `Show a detailed summary of a HAR file, including its version, creator, page count,
request count, total transfer size, timing percentiles, and distributions of status codes,
methods, domains, and content types.`,
	Example: `  har -f capture.har info
  har -f capture.har info --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)
		stats := h.Statistics()

		return internal.WriteOutput(cmd, stats, func() string {
			return formatInfoText(h, stats)
		}, nil)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

// formatInfoText formats a HAR summary as text.
func formatInfoText(h *har.Har, stats *har.HarStatistics) string {
	var sb strings.Builder

	// General information.
	sb.WriteString("HAR File Summary\n")
	sb.WriteString("=============\n")
	sb.WriteString(fmt.Sprintf("Version:       %s\n", h.Log.Version))
	if h.Log.Creator.Name != "" {
		sb.WriteString(fmt.Sprintf("Creator:       %s %s\n", h.Log.Creator.Name, h.Log.Creator.Version))
	}
	if h.Log.Browser.Name != "" {
		sb.WriteString(fmt.Sprintf("Browser:       %s %s\n", h.Log.Browser.Name, h.Log.Browser.Version))
	}
	sb.WriteString(fmt.Sprintf("Pages:         %d\n", len(h.Log.Pages)))
	sb.WriteString(fmt.Sprintf("Requests:      %d\n", stats.TotalRequests))

	// Transfer size.
	sb.WriteString(fmt.Sprintf("Transferred:   %s\n", internal.FormatBytes(int(stats.TotalTransferred))))
	sb.WriteString(fmt.Sprintf("Uncompressed:  %s\n", internal.FormatBytes(int(stats.TotalUncompressed))))

	// Timing statistics.
	sb.WriteString("\nTiming Statistics\n")
	sb.WriteString("--------\n")
	sb.WriteString(fmt.Sprintf("Total time:    %s\n", internal.FormatDuration(stats.TotalTime)))
	sb.WriteString(fmt.Sprintf("Average:       %s\n", internal.FormatDuration(stats.AvgTime)))
	sb.WriteString(fmt.Sprintf("Median:        %s\n", internal.FormatDuration(stats.MedianTime)))
	sb.WriteString(fmt.Sprintf("P95:           %s\n", internal.FormatDuration(stats.P95Time)))
	sb.WriteString(fmt.Sprintf("P99:           %s\n", internal.FormatDuration(stats.P99Time)))
	sb.WriteString(fmt.Sprintf("Slowest:       %s\n", internal.FormatDuration(stats.MaxTime)))
	sb.WriteString(fmt.Sprintf("Fastest:       %s\n", internal.FormatDuration(stats.MinTime)))

	// Errors and redirects.
	sb.WriteString(fmt.Sprintf("\nError requests: %d\n", stats.ErrorCount))
	sb.WriteString(fmt.Sprintf("Redirects:      %d\n", stats.RedirectCount))

	// Status code distribution.
	statusDist := h.StatusCodeDistribution()
	sb.WriteString("\nStatus Code Distribution\n")
	sb.WriteString("----------\n")
	for _, code := range sortedKeysInt(statusDist) {
		sb.WriteString(fmt.Sprintf("  %d: %d\n", code, statusDist[code]))
	}

	// Method distribution.
	methodDist := h.MethodDistribution()
	sb.WriteString("\nMethod Distribution\n")
	sb.WriteString("--------\n")
	for _, method := range sortedKeysStr(methodDist) {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", method, methodDist[method]))
	}

	// Domain distribution (top 10).
	sb.WriteString("\nDomain Distribution (Top 10)\n")
	sb.WriteString("----------------\n")
	topDomains := topN(stats.Domains, 10)
	for _, d := range topDomains {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", d.key, d.count))
	}

	// Content type distribution (top 10).
	contentDist := h.ContentTypeDistribution()
	sb.WriteString("\nContent Type Distribution (Top 10)\n")
	sb.WriteString("--------------------\n")
	topContentTypes := topN(contentDist, 10)
	for _, ct := range topContentTypes {
		sb.WriteString(fmt.Sprintf("  %s: %d\n", ct.key, ct.count))
	}

	return sb.String()
}

// keyValue is a key-value pair used for sorting.
type keyValue struct {
	key   string
	count int
}

// topN returns the n keys with the largest values in a map.
func topN(m map[string]int, n int) []keyValue {
	var items []keyValue
	for k, v := range m {
		items = append(items, keyValue{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})
	if len(items) > n {
		items = items[:n]
	}
	return items
}

// sortedKeysInt returns the map's integer keys in sorted order.
func sortedKeysInt(m map[int]int) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// sortedKeysStr returns the map's string keys in sorted order.
func sortedKeysStr(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
