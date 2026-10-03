package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// replayCmd replays HTTP requests recorded in a HAR file.
var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay HTTP requests from a HAR file",
	Long: `Re-execute HTTP requests recorded in a HAR file and display the responses.

Options include timeouts, redirects, and SSL verification. Dry-run mode previews
the requests without executing them.

Examples:
  har -f capture.har replay
  har -f capture.har replay --dry-run
  har -f capture.har replay --timeout 10s --skip-ssl
  har -f capture.har replay --index 0
  har -f capture.har replay --filter "api/users" --format json`,
	RunE: runReplay,
}

func init() {
	rootCmd.AddCommand(replayCmd)

	replayCmd.Flags().Duration("timeout", 30*time.Second, "Request timeout")
	replayCmd.Flags().Bool("no-follow-redirects", false, "Do not follow redirects")
	replayCmd.Flags().Int("max-redirects", 10, "Maximum number of redirects")
	replayCmd.Flags().Bool("skip-ssl", false, "Skip SSL certificate verification")
	replayCmd.Flags().StringSlice("header", nil, "Override request headers (format: name:value)")
	replayCmd.Flags().Int("index", -1, "Replay only the entry at the specified index")
	replayCmd.Flags().String("filter", "", "URL filter pattern (replay matching entries only)")
	replayCmd.Flags().Bool("dry-run", false, "Preview requests without executing them")
	replayCmd.Flags().String("save-har", "", "Save replay results as a new HAR file")
}

func runReplay(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	// Read options.
	timeout, _ := cmd.Flags().GetDuration("timeout")
	noFollowRedirects, _ := cmd.Flags().GetBool("no-follow-redirects")
	maxRedirects, _ := cmd.Flags().GetInt("max-redirects")
	skipSSL, _ := cmd.Flags().GetBool("skip-ssl")
	overrideHeadersSlice, _ := cmd.Flags().GetStringSlice("header")
	idx, _ := cmd.Flags().GetInt("index")
	filterPattern, _ := cmd.Flags().GetString("filter")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Parse header overrides.
	overrideHeaders := make(map[string]string)
	for _, h := range overrideHeadersSlice {
		name, value, err := parseColonKeyValue(h)
		if err != nil {
			return fmt.Errorf("invalid header argument '%s': %w", h, err)
		}
		overrideHeaders[name] = value
	}

	// Build replay options.
	opts := har.ReplayOptions{
		Timeout:         timeout,
		FollowRedirects: !noFollowRedirects,
		MaxRedirects:    maxRedirects,
		SkipSSLVerify:   skipSSL,
		OverrideHeaders: overrideHeaders,
	}

	// Select entries by index or filter.
	entries := selectReplayEntries(h, idx, filterPattern)

	// Dry-run mode displays the requests without replaying them.
	if dryRun {
		return internal.WriteOutput(cmd, formatDryRunEntries(entries), func() string {
			return formatDryRunText(entries)
		}, nil)
	}

	// Replay the requests.
	results, err := replayEntries(entries, opts)
	if err != nil {
		return fmt.Errorf("failed to replay requests: %w", err)
	}

	// Save replay results to a HAR file.
	saveHarPath, _ := cmd.Flags().GetString("save-har")
	if saveHarPath != "" {
		// Collect raw ReplayResults for ReplayResultsToHar
		replayResults := make([]*har.ReplayResult, 0)
		for i, entry := range entries {
			result, replayErr := entry.Replay(opts)
			if replayErr != nil {
				continue
			}
			_ = i
			replayResults = append(replayResults, result)
		}
		replayHar := har.ReplayResultsToHar(replayResults)
		if saveErr := replayHar.SaveToFile(saveHarPath, true); saveErr != nil {
			return fmt.Errorf("failed to save replay results as HAR: %w", saveErr)
		}
	}

	return internal.WriteOutput(cmd, results, func() string {
		return formatReplayResults(results)
	}, nil)
}

// replayEntryInfo contains entry information for dry-run mode and result display.
type replayEntryInfo struct {
	Index   int    `json:"index"`
	Method  string `json:"method"`
	URL     string `json:"url"`
	Headers int    `json:"headers"`
	HasBody bool   `json:"hasBody"`
}

// replayResultInfo contains replay results for JSON output.
type replayResultInfo struct {
	Index      int    `json:"index"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	StatusCode int    `json:"statusCode,omitempty"`
	Status     string `json:"status,omitempty"`
	Duration   string `json:"duration"`
	Error      string `json:"error,omitempty"`
}

// selectReplayEntries selects entries to replay by index or filter pattern.
func selectReplayEntries(h *har.Har, idx int, filter string) []har.Entries {
	if idx >= 0 {
		if idx >= len(h.Log.Entries) {
			return nil
		}
		return h.Log.Entries[idx : idx+1]
	}

	if filter != "" {
		var selected []har.Entries
		for i, entry := range h.Log.Entries {
			if strings.Contains(entry.Request.URL, filter) {
				_ = i
				selected = append(selected, entry)
			}
		}
		return selected
	}

	return h.Log.Entries
}

// formatDryRunEntries creates entry information for dry-run mode.
func formatDryRunEntries(entries []har.Entries) []replayEntryInfo {
	var infos []replayEntryInfo
	for i, entry := range entries {
		infos = append(infos, replayEntryInfo{
			Index:   i,
			Method:  entry.Request.Method,
			URL:     entry.Request.URL,
			Headers: len(entry.Request.Headers),
			HasBody: entry.Request.PostData != nil && entry.Request.PostData.Text != "",
		})
	}
	return infos
}

// formatDryRunText formats dry-run output as text.
func formatDryRunText(entries []har.Entries) string {
	var sb strings.Builder

	sb.WriteString("Dry run — requests to be replayed:\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	for i, entry := range entries {
		sb.WriteString(fmt.Sprintf("#%d %s %s\n", i, entry.Request.Method, entry.Request.URL))
		sb.WriteString(fmt.Sprintf("   Headers: %d", len(entry.Request.Headers)))
		if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
			sb.WriteString(", body present")
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString(fmt.Sprintf("%d request(s) to replay\n", len(entries)))
	return sb.String()
}

// replayEntries replays the entries.
func replayEntries(entries []har.Entries, opts har.ReplayOptions) ([]replayResultInfo, error) {
	results := make([]replayResultInfo, len(entries))
	var firstErr error

	for i, entry := range entries {
		result, err := entry.Replay(opts)
		info := replayResultInfo{
			Index:    i,
			Method:   entry.Request.Method,
			URL:      entry.Request.URL,
			Duration: result.Duration.String(),
		}

		if err != nil {
			info.Error = err.Error()
			if firstErr == nil {
				firstErr = err
			}
		} else if result.Response != nil {
			info.StatusCode = result.Response.StatusCode
			info.Status = result.Response.Status
			// Read and discard the response body to release the connection.
			_, _ = io.Copy(io.Discard, result.Response.Body)
			result.Response.Body.Close()
		}

		results[i] = info
	}

	return results, firstErr
}

// formatReplayResults formats replay results as text.
func formatReplayResults(results []replayResultInfo) string {
	var sb strings.Builder

	sb.WriteString("Replay Results\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	successCount := 0
	failCount := 0

	for _, r := range results {
		status := "OK"
		if r.Error != "" {
			status = "FAIL"
			failCount++
		} else {
			successCount++
		}

		sb.WriteString(fmt.Sprintf("#%d %s %s\n", r.Index, r.Method, r.URL))

		if r.StatusCode > 0 {
			sb.WriteString(fmt.Sprintf("   Status: %d %s  Duration: %s\n", r.StatusCode, r.Status, r.Duration))
		} else {
			sb.WriteString(fmt.Sprintf("   Status: %s  Duration: %s\n", status, r.Duration))
		}

		if r.Error != "" {
			sb.WriteString(fmt.Sprintf("   Error: %s\n", r.Error))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Total: %d succeeded, %d failed\n", successCount, failCount))
	return sb.String()
}
