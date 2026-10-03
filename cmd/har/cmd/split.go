package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// splitCmd splits a HAR file.
var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "Split a HAR file",
	Long: `Split a HAR file into multiple smaller files using various criteria.

Supported split modes:
  --by page     Split by page reference (pageref)
  --by domain   Split by request domain
  --by time     Split by time interval (with --interval)
  --by size     Split by entry count (with --max-entries)
  --by status   Split by status code range (2xx/3xx/4xx/5xx)
  --by method   Split by HTTP method

Examples:
  har split -f capture.har --by domain                     # Split by domain
  har split -f capture.har --by time --interval 30m        # Split every 30 minutes
  har split -f capture.har --by size --max-entries 50      # Split every 50 entries
  har split -f capture.har --by status -o result           # Use "result" as the output prefix`,
	RunE: runSplit,
}

func init() {
	rootCmd.AddCommand(splitCmd)

	splitCmd.Flags().String("by", "", "Split mode (page/domain/time/size/status/method)")
	splitCmd.Flags().Duration("interval", 1*time.Hour, "Time interval (with --by time)")
	splitCmd.Flags().Int("max-entries", 100, "Maximum entries per group (with --by size)")
	splitCmd.Flags().StringP("output", "o", "split", "Output file prefix")
}

// runSplit executes the split command.
func runSplit(cmd *cobra.Command, args []string) error {
	// Check that the required --by flag is set.
	by, _ := cmd.Flags().GetString("by")
	if by == "" {
		return fmt.Errorf("--by is required (page/domain/time/size/status/method)")
	}

	// Load the HAR file.
	h := internal.LoadHar(cmd, args)

	// Get the output prefix (prefer the local -o flag over global --output).
	prefix, _ := cmd.Flags().GetString("output")
	if prefix == "" {
		prefix = "split"
	}

	// Split according to the selected mode.
	var fileCount int
	var err error

	switch by {
	case "page":
		fileCount, err = splitByPage(h, prefix)
	case "domain":
		fileCount, err = splitByDomain(h, prefix)
	case "time":
		interval, _ := cmd.Flags().GetDuration("interval")
		fileCount, err = splitByTime(h, prefix, interval)
	case "size":
		maxEntries, _ := cmd.Flags().GetInt("max-entries")
		fileCount, err = splitBySize(h, prefix, maxEntries)
	case "status":
		fileCount, err = splitByStatus(h, prefix)
	case "method":
		fileCount, err = splitByMethod(h, prefix)
	default:
		return fmt.Errorf("unsupported split mode: %s (choose from page/domain/time/size/status/method)", by)
	}

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Split into %d file(s) (prefix: %s)\n", fileCount, prefix)
	return nil
}

// splitByPage splits by page reference.
func splitByPage(h *har.Har, prefix string) (int, error) {
	parts := h.SplitByPage()
	return writeSplitMap(parts, prefix, "page")
}

// splitByDomain splits by domain.
func splitByDomain(h *har.Har, prefix string) (int, error) {
	parts := h.SplitByDomain()
	return writeSplitMap(parts, prefix, "domain")
}

// splitByTime splits by time interval.
func splitByTime(h *har.Har, prefix string, interval time.Duration) (int, error) {
	parts := h.SplitByTimeRange(interval)
	return writeSplitSlice(parts, prefix, "time")
}

// splitBySize splits by entry count.
func splitBySize(h *har.Har, prefix string, maxEntries int) (int, error) {
	parts := h.SplitBySize(maxEntries)
	return writeSplitSlice(parts, prefix, "size")
}

// splitByStatus splits by status code range.
func splitByStatus(h *har.Har, prefix string) (int, error) {
	parts := h.SplitByStatusCode()
	return writeSplitMap(parts, prefix, "status")
}

// splitByMethod splits by HTTP method.
func splitByMethod(h *har.Har, prefix string) (int, error) {
	parts := h.SplitByMethod()
	return writeSplitMap(parts, prefix, "method")
}

// writeSplitMap writes map-based split results to files.
func writeSplitMap(parts map[string]*har.Har, prefix, kind string) (int, error) {
	for key, harData := range parts {
		// Sanitize special characters in the key.
		safeKey := sanitizeFilename(key)
		if safeKey == "" {
			safeKey = "unnamed"
		}
		filename := fmt.Sprintf("%s_%s_%s.har", prefix, kind, safeKey)
		if err := writeHarToFile(harData, filename); err != nil {
			return 0, err
		}
	}
	return len(parts), nil
}

// writeSplitSlice writes slice-based split results to files.
func writeSplitSlice(parts []*har.Har, prefix, kind string) (int, error) {
	for i, harData := range parts {
		filename := fmt.Sprintf("%s_%s_%03d.har", prefix, kind, i+1)
		if err := writeHarToFile(harData, filename); err != nil {
			return 0, err
		}
	}
	return len(parts), nil
}

// writeHarToFile writes HAR data to a file.
func writeHarToFile(h *har.Har, filename string) error {
	output, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON serialization failed: %w", err)
	}
	output = append(output, '\n')

	// Ensure the directory exists.
	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("unable to create directory '%s': %w", dir, err)
		}
	}

	if err := os.WriteFile(filename, output, 0644); err != nil {
		return fmt.Errorf("unable to write file '%s': %w", filename, err)
	}

	fmt.Fprintf(os.Stderr, "  Wrote: %s (%d entries)\n", filename, len(h.Log.Entries))
	return nil
}

// sanitizeFilename replaces special characters in a file name.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}
