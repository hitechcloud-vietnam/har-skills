package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// findCmd searches HAR entries.
var findCmd = &cobra.Command{
	Use:   "find [pattern]",
	Short: "Search HAR entries",
	Long: `Search HAR entries by URL, status code, or conditions. Supports regex URL matching,
status code range, content type, domain, request/response headers, cookies, resource type,
time range, server IP, connection ID, cache hits, slow/fast/largest requests, etc.`,
	Example: `  har -f capture.har find "api/users"
  har -f capture.har find --regex "api/v[0-9]+"
  har -f capture.har find --errors
  har -f capture.har find --redirects
  har -f capture.har find --slow 1000
  har -f capture.har find --fastest 5
  har -f capture.har find --slowest 5
  har -f capture.har find --largest 5
  har -f capture.har find --status-code 404
  har -f capture.har find --method GET --domain example.com
  har -f capture.har find --response-header "Content-Type:application/json"
  har -f capture.har find --cookie "session_id"
  har -f capture.har find --start-time "2024-01-01T00:00:00Z" --end-time "2024-12-31T23:59:59Z"
  har -f capture.har find --server-ip "10.0.0.1"
  har -f capture.har find --cache-hits
  har -f capture.har find --connection "ABC123"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)

		// Read all filter parameters.
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		useRegex, _ := cmd.Flags().GetBool("regex")
		method, _ := cmd.Flags().GetString("method")
		statusCode, _ := cmd.Flags().GetInt("status-code")
		statusMin, _ := cmd.Flags().GetInt("status-min")
		statusMax, _ := cmd.Flags().GetInt("status-max")
		contentType, _ := cmd.Flags().GetString("content-type")
		domain, _ := cmd.Flags().GetString("domain")
		headers, _ := cmd.Flags().GetStringSlice("header")
		responseHeaders, _ := cmd.Flags().GetStringSlice("response-header")
		cookieName, _ := cmd.Flags().GetString("cookie")
		resourceType, _ := cmd.Flags().GetString("resource-type")
		errors, _ := cmd.Flags().GetBool("errors")
		redirects, _ := cmd.Flags().GetBool("redirects")
		slow, _ := cmd.Flags().GetFloat64("slow")
		cacheHits, _ := cmd.Flags().GetBool("cache-hits")
		serverIP, _ := cmd.Flags().GetString("server-ip")
		connection, _ := cmd.Flags().GetString("connection")
		startTimeStr, _ := cmd.Flags().GetString("start-time")
		endTimeStr, _ := cmd.Flags().GetString("end-time")
		slowest, _ := cmd.Flags().GetInt("slowest")
		fastest, _ := cmd.Flags().GetInt("fastest")
		largest, _ := cmd.Flags().GetInt("largest")
		limit, _ := cmd.Flags().GetInt("limit")

		// Build filter options.
		opts := []har.FilterOption{}

		// Match URL patterns.
		if pattern != "" {
			if useRegex {
				opts = append(opts, har.WithFilterURL(pattern))
				opts = append(opts, har.WithFilterRegex())
			} else {
				opts = append(opts, har.WithFilterURL(pattern))
			}
		}

		// Filter by method.
		if method != "" {
			opts = append(opts, har.WithFilterMethod(method))
		}

		// Filter by status code.
		if statusCode > 0 {
			opts = append(opts, har.WithFilterStatusCode(statusCode))
		}

		// Filter by status code range.
		if statusMin > 0 || statusMax > 0 {
			opts = append(opts, har.WithFilterStatusCodeRange(statusMin, statusMax))
		}

		// Filter by content type.
		if contentType != "" {
			opts = append(opts, har.WithFilterContentType(contentType))
		}

		// Filter errors.
		if errors {
			opts = append(opts, har.WithFilterHasError())
		}

		// Filter by resource type.
		if resourceType != "" {
			opts = append(opts, har.WithFilterResourceType(resourceType))
		}

		// Filter slow requests.
		if slow > 0 {
			opts = append(opts, har.WithFilterDuration(slow, 0))
		}

		// Filter by request headers.
		for _, h := range headers {
			parts := strings.SplitN(h, ":", 2)
			name := parts[0]
			value := ""
			if len(parts) > 1 {
				value = strings.TrimSpace(parts[1])
			}
			opts = append(opts, har.WithFilterHeader(name, value))
		}

		// Apply filters.
		var result *har.FilterResult
		if len(opts) > 0 {
			result = h.FilterWith(opts...)
		} else {
			result = &har.FilterResult{Entries: h.Log.Entries}
		}

		// Filter by domain (handled separately).
		if domain != "" {
			var filtered []har.Entries
			for _, entry := range result.Entries {
				if d := har.ExtractDomain(entry.Request.URL); d == domain {
					filtered = append(filtered, entry)
				}
			}
			result = &har.FilterResult{Entries: filtered}
		}

		// Filter by response headers.
		if len(responseHeaders) > 0 {
			for _, rh := range responseHeaders {
				parts := strings.SplitN(rh, ":", 2)
				name := parts[0]
				value := ""
				if len(parts) > 1 {
					value = strings.TrimSpace(parts[1])
				}
				respResult := h.FindByResponseHeader(name, value)
				result = intersectResults(result, respResult)
			}
		}

		// Filter by cookie.
		if cookieName != "" {
			cookieResult := h.FindByCookie(cookieName)
			result = intersectResults(result, cookieResult)
		}

		// Filter by time range.
		if startTimeStr != "" || endTimeStr != "" {
			startTime := time.Time{}
			endTime := time.Now()
			if startTimeStr != "" {
				t, err := time.Parse(time.RFC3339, startTimeStr)
				if err != nil {
					return fmt.Errorf("invalid start-time format (use RFC3339): %w", err)
				}
				startTime = t
			}
			if endTimeStr != "" {
				t, err := time.Parse(time.RFC3339, endTimeStr)
				if err != nil {
					return fmt.Errorf("invalid end-time format (use RFC3339): %w", err)
				}
				endTime = t
			}
			timeResult := h.FindByTimeRange(startTime, endTime)
			result = intersectResults(result, timeResult)
		}

		// Filter by server IP.
		if serverIP != "" {
			ipResult := h.FindByServerIP(serverIP)
			result = intersectResults(result, ipResult)
		}

		// Filter by connection.
		if connection != "" {
			connResult := h.FindByConnection(connection)
			result = intersectResults(result, connResult)
		}

		// Filter cache hits.
		if cacheHits {
			cacheResult := h.FindCacheHits()
			result = intersectResults(result, cacheResult)
		}

		// Filter redirects.
		if redirects {
			redirectResult := h.FindRedirects()
			result = intersectResults(result, redirectResult)
		}

		// Slowest N requests
		if slowest > 0 {
			slowestEntries := h.SlowestRequests(slowest)
			result = intersectResults(result, &har.FilterResult{Entries: slowestEntries})
		}

		// Fastest N requests
		if fastest > 0 {
			fastestEntries := h.FastestRequests(fastest)
			result = intersectResults(result, &har.FilterResult{Entries: fastestEntries})
		}

		// Largest N responses
		if largest > 0 {
			largestEntries := h.LargestResponses(largest)
			result = intersectResults(result, &har.FilterResult{Entries: largestEntries})
		}

		// Limit the number of entries.
		if limit > 0 {
			result.Limit(limit)
		}

		return internal.WriteOutput(cmd, buildListJSON(result), func() string {
			return formatFindTable(result)
		}, nil)
	},
}

// intersectResults returns the intersection of two FilterResults.
func intersectResults(a, b *har.FilterResult) *har.FilterResult {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	bSet := make(map[string]bool)
	for _, e := range b.Entries {
		bSet[e.Request.URL+e.Request.Method] = true
	}
	var filtered []har.Entries
	for _, entry := range a.Entries {
		if bSet[entry.Request.URL+entry.Request.Method] {
			filtered = append(filtered, entry)
		}
	}
	return &har.FilterResult{Entries: filtered}
}

func init() {
	rootCmd.AddCommand(findCmd)

	// URL and pattern
	findCmd.Flags().Bool("regex", false, "Use regex for URL pattern matching")
	// Method and status
	findCmd.Flags().String("method", "", "Filter by HTTP method")
	findCmd.Flags().Int("status-code", 0, "Filter by exact status code")
	findCmd.Flags().Int("status-min", 0, "Minimum status code (range filter)")
	findCmd.Flags().Int("status-max", 0, "Maximum status code (range filter)")
	// Content and type
	findCmd.Flags().String("content-type", "", "Filter by content type")
	findCmd.Flags().String("resource-type", "", "Filter by resource type (document, script, stylesheet, image, font, xhr, etc.)")
	// Domain and network
	findCmd.Flags().String("domain", "", "Filter by domain name")
	findCmd.Flags().String("server-ip", "", "Filter by server IP address")
	findCmd.Flags().String("connection", "", "Filter by connection ID")
	// Headers
	findCmd.Flags().StringSlice("header", nil, "Filter by request header (format: name or name:value)")
	findCmd.Flags().StringSlice("response-header", nil, "Filter by response header (format: name or name:value)")
	// Cookies
	findCmd.Flags().String("cookie", "", "Filter entries containing a cookie by name")
	// Time
	findCmd.Flags().String("start-time", "", "Filter entries after this time (RFC3339 format, e.g. 2024-01-01T00:00:00Z)")
	findCmd.Flags().String("end-time", "", "Filter entries before this time (RFC3339 format)")
	findCmd.Flags().Float64("slow", 0, "Find slow requests (minimum duration in ms)")
	findCmd.Flags().Int("slowest", 0, "Find the N slowest requests")
	findCmd.Flags().Int("fastest", 0, "Find the N fastest requests")
	findCmd.Flags().Int("largest", 0, "Find the N largest responses by size")
	// Special filters
	findCmd.Flags().Bool("errors", false, "Find all error requests (4xx/5xx)")
	findCmd.Flags().Bool("redirects", false, "Find all redirect requests (3xx)")
	findCmd.Flags().Bool("cache-hits", false, "Find requests with cache hits")
	// Output
	findCmd.Flags().IntP("limit", "n", 0, "Limit output to N entries (0=all)")
}

// formatFindTable formats search results as a tabwriter table.
func formatFindTable(result *har.FilterResult) string {
	var sb tabWriterBuf

	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "INDEX\tMETHOD\tSTATUS\tSIZE\tTIME\tURL\n")

	for i, entry := range result.Entries {
		size := internal.FormatBytes(entry.Response.Content.Size)
		dur := internal.FormatDuration(entry.Time)
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
			i, entry.Request.Method, entry.Response.Status, size, dur, entry.Request.URL)
	}
	w.Flush()

	return sb.String()
}
