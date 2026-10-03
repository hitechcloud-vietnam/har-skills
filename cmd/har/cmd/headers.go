package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// headersCmd displays request and response headers.
var headersCmd = &cobra.Command{
	Use:   "headers [url-pattern]",
	Short: "Show request and response headers",
	Long: `Show request and response headers for matching entries.
Choose request headers, response headers, or filter by header name.`,
	Example: `  har -f capture.har headers
  har -f capture.har headers "api/users"
  har -f capture.har headers --request
  har -f capture.har headers --response --name content-type`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)

		// Read the arguments.
		urlPattern := ""
		if len(args) > 0 {
			urlPattern = args[0]
		}
		showRequest, _ := cmd.Flags().GetBool("request")
		showResponse, _ := cmd.Flags().GetBool("response")
		headerName, _ := cmd.Flags().GetString("name")
		limit, _ := cmd.Flags().GetInt("limit")

		// Show both request and response headers by default.
		if !showRequest && !showResponse {
			showRequest = true
			showResponse = true
		}

		// Filter entries.
		var entries []har.Entries
		for _, entry := range h.Log.Entries {
			if urlPattern != "" && !strings.Contains(entry.Request.URL, urlPattern) {
				continue
			}
			entries = append(entries, entry)
		}

		// Limit the number of entries.
		if limit > 0 && limit < len(entries) {
			entries = entries[:limit]
		}

		return internal.WriteOutput(cmd, buildHeadersJSON(entries, showRequest, showResponse, headerName), func() string {
			return formatHeadersText(entries, showRequest, showResponse, headerName)
		}, nil)
	},
}

func init() {
	rootCmd.AddCommand(headersCmd)

	headersCmd.Flags().Bool("request", false, "Show request headers only")
	headersCmd.Flags().Bool("response", false, "Show response headers only")
	headersCmd.Flags().String("name", "", "Filter by header name (case-insensitive)")
	headersCmd.Flags().IntP("limit", "n", 1, "Number of entries to show (default: 1)")
}

// headerEntry contains header information for JSON output.
type headerEntry struct {
	Index    int               `json:"index"`
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Status   int               `json:"status"`
	Request  map[string]string `json:"requestHeaders,omitempty"`
	Response map[string]string `json:"responseHeaders,omitempty"`
}

// buildHeadersJSON builds the JSON output.
func buildHeadersJSON(entries []har.Entries, showRequest, showResponse bool, headerName string) []headerEntry {
	result := make([]headerEntry, 0, len(entries))
	for i, entry := range entries {
		he := headerEntry{
			Index:  i,
			URL:    entry.Request.URL,
			Method: entry.Request.Method,
			Status: entry.Response.Status,
		}
		if showRequest {
			he.Request = filterHeaders(entry.Request.Headers, headerName)
		}
		if showResponse {
			he.Response = filterHeaders(entry.Response.Headers, headerName)
		}
		result = append(result, he)
	}
	return result
}

// filterHeaders filters headers by name.
func filterHeaders(headers []har.Headers, name string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		if name != "" && !strings.EqualFold(h.Name, name) {
			continue
		}
		result[h.Name] = h.Value
	}
	return result
}

// formatHeadersText formats header information as text.
func formatHeadersText(entries []har.Entries, showRequest, showResponse bool, headerName string) string {
	var sb strings.Builder

	for i, entry := range entries {
		sb.WriteString(fmt.Sprintf("=== Entry #%d ===\n", i))
		sb.WriteString(fmt.Sprintf("URL: %s %s\n", entry.Request.Method, entry.Request.URL))
		sb.WriteString(fmt.Sprintf("Status: %d %s\n", entry.Response.Status, entry.Response.StatusText))

		if showRequest {
			sb.WriteString("\nRequest Headers:\n")
			for _, h := range entry.Request.Headers {
				if headerName != "" && !strings.EqualFold(h.Name, headerName) {
					continue
				}
				sb.WriteString(fmt.Sprintf("  %s: %s\n", h.Name, h.Value))
			}
		}

		if showResponse {
			sb.WriteString("\nResponse Headers:\n")
			for _, h := range entry.Response.Headers {
				if headerName != "" && !strings.EqualFold(h.Name, headerName) {
					continue
				}
				sb.WriteString(fmt.Sprintf("  %s: %s\n", h.Name, h.Value))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
