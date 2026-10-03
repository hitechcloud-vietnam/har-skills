package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// timingCmd analyzes request timing breakdowns.
var timingCmd = &cobra.Command{
	Use:   "timing",
	Short: "Analyze request timing breakdowns",
	Long: `Analyze request timing in a HAR file, including blocked, DNS lookup,
TCP connection, SSL handshake, send, wait, and receive durations.
Entries can be sorted or limited, and summary statistics are available.`,
	Example: `  har -f capture.har timing
  har -f capture.har timing --sort wait --limit 10
  har -f capture.har timing --summary
  har -f capture.har timing --filter "api/users"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)

		// Read the arguments.
		filter, _ := cmd.Flags().GetString("filter")
		sortBy, _ := cmd.Flags().GetString("sort")
		limit, _ := cmd.Flags().GetInt("limit")
		showSummary, _ := cmd.Flags().GetBool("summary")

		// Filter entries.
		var entries []har.Entries
		for _, entry := range h.Log.Entries {
			if filter != "" && !strings.Contains(entry.Request.URL, filter) {
				continue
			}
			entries = append(entries, entry)
		}

		// Sort the entries.
		sortEntries(entries, sortBy)

		// Limit the number of entries.
		if limit > 0 && limit < len(entries) {
			entries = entries[:limit]
		}

		// Summary mode.
		if showSummary {
			timingsSummary := h.TimingStatistics()
			return internal.WriteOutput(cmd, timingsSummary, func() string {
				return formatTimingSummary(timingsSummary)
			}, nil)
		}

		return internal.WriteOutput(cmd, buildTimingJSON(entries), func() string {
			return formatTimingTable(entries)
		}, nil)
	},
}

func init() {
	rootCmd.AddCommand(timingCmd)

	timingCmd.Flags().String("filter", "", "URL filter string")
	timingCmd.Flags().String("sort", "time", "Sort by (time, wait, dns, connect)")
	timingCmd.Flags().IntP("limit", "n", 0, "Maximum number of entries (0 = all)")
	timingCmd.Flags().Bool("summary", false, "Show summary statistics")
}

// timingEntry contains timing information for JSON output.
type timingEntry struct {
	URL     string  `json:"url"`
	Total   float64 `json:"total"`
	Blocked float64 `json:"blocked"`
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	SSL     float64 `json:"ssl"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

// sortEntries sorts entries by the specified field.
func sortEntries(entries []har.Entries, sortBy string) {
	switch sortBy {
	case "wait":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timings.Wait > entries[j].Timings.Wait
		})
	case "dns":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timings.DNS > entries[j].Timings.DNS
		})
	case "connect":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timings.Connect > entries[j].Timings.Connect
		})
	default: // time
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Time > entries[j].Time
		})
	}
}

// buildTimingJSON builds the JSON output.
func buildTimingJSON(entries []har.Entries) []timingEntry {
	result := make([]timingEntry, len(entries))
	for i, entry := range entries {
		result[i] = timingEntry{
			URL:     entry.Request.URL,
			Total:   entry.Time,
			Blocked: entry.Timings.Blocked,
			DNS:     entry.Timings.DNS,
			Connect: entry.Timings.Connect,
			SSL:     entry.Timings.Ssl,
			Send:    entry.Timings.Send,
			Wait:    entry.Timings.Wait,
			Receive: entry.Timings.Receive,
		}
	}
	return result
}

// formatTimingTable formats timing information as a tabwriter table.
func formatTimingTable(entries []har.Entries) string {
	var sb tabWriterBuf

	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "URL\tTOTAL\tBLOCKED\tDNS\tCONNECT\tSSL\tSEND\tWAIT\tRECEIVE\n")

	for _, entry := range entries {
		url := entry.Request.URL
		// Truncate long URLs.
		if len(url) > 60 {
			url = url[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			url,
			internal.FormatDuration(entry.Time),
			formatTimingValue(entry.Timings.Blocked),
			formatTimingValue(entry.Timings.DNS),
			formatTimingValue(entry.Timings.Connect),
			formatTimingValue(entry.Timings.Ssl),
			formatTimingValue(entry.Timings.Send),
			formatTimingValue(entry.Timings.Wait),
			formatTimingValue(entry.Timings.Receive),
		)
	}
	w.Flush()

	return sb.String()
}

// formatTimingSummary formats timing summary information.
func formatTimingSummary(summary *har.TimingsSummary) string {
	var sb strings.Builder

	sb.WriteString("Timing Summary\n")
	sb.WriteString("========\n")
	sb.WriteString(fmt.Sprintf("Average blocked: %s\n", internal.FormatDuration(summary.AvgBlocked)))
	sb.WriteString(fmt.Sprintf("Average DNS:     %s\n", internal.FormatDuration(summary.AvgDNS)))
	sb.WriteString(fmt.Sprintf("Average connect: %s\n", internal.FormatDuration(summary.AvgConnect)))
	sb.WriteString(fmt.Sprintf("Average SSL:     %s\n", internal.FormatDuration(summary.AvgSSL)))
	sb.WriteString(fmt.Sprintf("Average send:    %s\n", internal.FormatDuration(summary.AvgSend)))
	sb.WriteString(fmt.Sprintf("Average wait:    %s\n", internal.FormatDuration(summary.AvgWait)))
	sb.WriteString(fmt.Sprintf("Average receive: %s\n", internal.FormatDuration(summary.AvgReceive)))

	sb.WriteString("\nMaximums\n")
	sb.WriteString("------\n")
	sb.WriteString(fmt.Sprintf("Maximum blocked: %s\n", internal.FormatDuration(summary.MaxBlocked)))
	sb.WriteString(fmt.Sprintf("Maximum DNS:     %s\n", internal.FormatDuration(summary.MaxDNS)))
	sb.WriteString(fmt.Sprintf("Maximum connect: %s\n", internal.FormatDuration(summary.MaxConnect)))
	sb.WriteString(fmt.Sprintf("Maximum SSL:     %s\n", internal.FormatDuration(summary.MaxSSL)))
	sb.WriteString(fmt.Sprintf("Maximum send:    %s\n", internal.FormatDuration(summary.MaxSend)))
	sb.WriteString(fmt.Sprintf("Maximum wait:    %s\n", internal.FormatDuration(summary.MaxWait)))
	sb.WriteString(fmt.Sprintf("Maximum receive: %s\n", internal.FormatDuration(summary.MaxReceive)))

	return sb.String()
}

// formatTimingValue formats a timing value, displaying negative values as "-".
func formatTimingValue(v float64) string {
	if v < 0 {
		return "-"
	}
	return internal.FormatDuration(v)
}
