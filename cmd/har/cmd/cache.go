package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// cacheCmd analyzes cache headers in a HAR file.
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Analyze HAR cache headers",
	Long: `Analyze cache-related response headers in a HAR file and assess the cacheability of each request.

Checks headers such as Cache-Control, ETag, Last-Modified, and Vary,
then outputs a cacheability assessment for each entry.

Examples:
  har -f capture.har cache
  har -f capture.har cache --non-cacheable
  har -f capture.har cache --url "https://api.example.com/data"
  har -f capture.har cache --format json`,
	RunE: runCache,
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	cacheCmd.Flags().Bool("non-cacheable", false, "Show only non-cacheable entries")
	cacheCmd.Flags().String("url", "", "Show the cache assessment for the specified URL only")
}

func runCache(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	report := h.CacheAnalysis()

	// Apply filters.
	showNonCacheable, _ := cmd.Flags().GetBool("non-cacheable")
	specificURL, _ := cmd.Flags().GetString("url")

	if showNonCacheable {
		report.Assessments = report.NonCacheableEntries()
	}

	if specificURL != "" {
		assessment := report.FindByURL(specificURL)
		if assessment == nil {
			return fmt.Errorf("no cache assessment found for URL '%s'", specificURL)
		}
		// Keep only the matching assessment.
		report.Assessments = []har.CacheEntryAssessment{*assessment}
	}

	return internal.WriteOutput(cmd, report, func() string {
		return formatCacheReport(report)
	}, nil)
}

// formatCacheReport formats the cache analysis report as text.
func formatCacheReport(report *har.CacheReport) string {
	var sb strings.Builder

	sb.WriteString("Cache Analysis Report\n")
	sb.WriteString("============\n")
	sb.WriteString(fmt.Sprintf("Cacheable: %d / Non-cacheable: %d / Cache efficiency: %.1f%%\n\n",
		report.CacheableCount, report.NonCacheableCount, report.CacheEfficiency*100))

	if len(report.Assessments) == 0 {
		sb.WriteString("No cache assessment data.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("%-4s %-50s %-10s %-10s %-10s %-6s\n",
		"#", "URL", "Cacheable", "Type", "Max-Age", "ETag"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	for _, a := range report.Assessments {
		cacheable := "No"
		if a.Cacheable {
			cacheable = "Yes"
		}
		hasETag := "No"
		if a.HasETag {
			hasETag = "Yes"
		}
		maxAge := "N/A"
		if a.MaxAge > 0 {
			maxAge = fmt.Sprintf("%.0fs", a.MaxAge.Seconds())
		}
		urlDisplay := a.URL
		if len(urlDisplay) > 50 {
			urlDisplay = urlDisplay[:47] + "..."
		}

		sb.WriteString(fmt.Sprintf("%-4d %-50s %-10s %-10s %-10s %-6s\n",
			a.EntryIndex, urlDisplay, cacheable, a.CacheType, maxAge, hasETag))
	}

	return sb.String()
}
