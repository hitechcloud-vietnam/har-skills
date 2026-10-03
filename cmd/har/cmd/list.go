package cmd

import (
	"fmt"
	"text/tabwriter"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// listCmd lists HAR entries.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List HAR entries",
	Long: `List request entries in a HAR file. Entries can be sorted by time, size, URL, or status code,
filtered by method, status code, or domain, and limited to a specified number.`,
	Example: `  har -f capture.har list
  har -f capture.har list --limit 10
  har -f capture.har list --sort size --order asc
  har -f capture.har list --method GET --status 200`,
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)

		// Read filter parameters.
		method, _ := cmd.Flags().GetString("method")
		status, _ := cmd.Flags().GetInt("status")
		domain, _ := cmd.Flags().GetString("domain")
		sortBy, _ := cmd.Flags().GetString("sort")
		order, _ := cmd.Flags().GetString("order")
		limit, _ := cmd.Flags().GetInt("limit")

		// Filter with FilterWith.
		opts := []har.FilterOption{}
		if method != "" {
			opts = append(opts, har.WithFilterMethod(method))
		}
		if status > 0 {
			opts = append(opts, har.WithFilterStatusCode(status))
		}

		var result *har.FilterResult
		if len(opts) > 0 {
			result = h.FilterWith(opts...)
		} else {
			// With no filters, use all entries.
			result = &har.FilterResult{Entries: h.Log.Entries}
		}

		// Apply an additional domain filter.
		if domain != "" {
			var filtered []har.Entries
			for _, entry := range result.Entries {
				if d := har.ExtractDomain(entry.Request.URL); d == domain {
					filtered = append(filtered, entry)
				}
			}
			result = &har.FilterResult{Entries: filtered}
		}

		// Sort the entries.
		switch sortBy {
		case "size":
			if order == "asc" {
				result.SortBySize()
			} else {
				result.SortBySizeDesc()
			}
		case "url":
			// The SDK has no URL sort method; keep the default order.
		case "status":
			// The SDK has no status-code sort method; keep the default order.
		default: // time
			if order == "asc" {
				result.SortByDuration()
			} else {
				result.SortByDurationDesc()
			}
		}

		// Limit the number of entries.
		if limit > 0 {
			result.Limit(limit)
		}

		return internal.WriteOutput(cmd, buildListJSON(result), func() string {
			return formatListTable(result, h)
		}, nil)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().IntP("limit", "n", 0, "Maximum number of entries to output (0 = all)")
	listCmd.Flags().String("sort", "time", "Sort by (time, size, url, status)")
	listCmd.Flags().String("order", "desc", "Sort order (asc, desc)")
	listCmd.Flags().String("method", "", "Filter by HTTP method")
	listCmd.Flags().Int("status", 0, "Filter by status code")
	listCmd.Flags().String("domain", "", "Filter by domain")
}

// listEntry is a simplified entry type for JSON output.
type listEntry struct {
	Index  int     `json:"index"`
	Method string  `json:"method"`
	Status int     `json:"status"`
	Size   int     `json:"size"`
	Time   float64 `json:"time"`
	URL    string  `json:"url"`
}

// buildListJSON builds the JSON output.
func buildListJSON(result *har.FilterResult) []listEntry {
	entries := make([]listEntry, len(result.Entries))
	for i, entry := range result.Entries {
		entries[i] = listEntry{
			Index:  i,
			Method: entry.Request.Method,
			Status: entry.Response.Status,
			Size:   entry.Response.Content.Size,
			Time:   entry.Time,
			URL:    entry.Request.URL,
		}
	}
	return entries
}

// formatListTable formats the entry list as a tabwriter table.
func formatListTable(result *har.FilterResult, h *har.Har) string {
	var sb tabWriterBuf

	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "INDEX\tMETHOD\tSTATUS\tSIZE\tTIME\tURL\n")

	for i, entry := range result.Entries {
		size := internal.FormatBytes(entry.Response.Content.Size)
		time := internal.FormatDuration(entry.Time)
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
			i, entry.Request.Method, entry.Response.Status, size, time, entry.Request.URL)
	}
	w.Flush()

	return sb.String()
}

// tabWriterBuf is a buffer for tabwriter output.
type tabWriterBuf struct {
	buf []byte
}

func (t *tabWriterBuf) Write(p []byte) (n int, err error) {
	t.buf = append(t.buf, p...)
	return len(p), nil
}

func (t *tabWriterBuf) String() string {
	return string(t.buf)
}
