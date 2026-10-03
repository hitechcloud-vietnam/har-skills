package cmd

import (
	"fmt"
	"os"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// extractCmd extracts response content.
var extractCmd = &cobra.Command{
	Use:   "extract [url-pattern]",
	Short: "Extract response content",
	Long: `Extract response content from matching entries. Entries can be selected by URL pattern or index.
Base64-encoded and gzip/deflate-compressed content can be decoded automatically.
Content is written to stdout by default, or to a file with --output.`,
	Example: `  har -f capture.har extract
  har -f capture.har extract "api/users"
  har -f capture.har extract --index 0
  har -f capture.har extract --all --decode
  har -f capture.har extract --index 3 -o response.json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		h := internal.LoadHar(cmd, args)

		// Read the arguments.
		urlPattern := ""
		if len(args) > 0 {
			urlPattern = args[0]
		}
		entryIndex, _ := cmd.Flags().GetInt("index")
		decode, _ := cmd.Flags().GetBool("decode")
		extractAll, _ := cmd.Flags().GetBool("all")

		// Extract by index.
		if entryIndex >= 0 && entryIndex < len(h.Log.Entries) {
			return extractSingleEntry(cmd, &h.Log.Entries[entryIndex], decode)
		}

		// Filter matching entries.
		var entries []har.Entries
		for _, entry := range h.Log.Entries {
			if urlPattern != "" && !strings.Contains(entry.Request.URL, urlPattern) {
				continue
			}
			entries = append(entries, entry)
		}

		// Extract all matches or only the first.
		if extractAll {
			return extractMultipleEntries(cmd, entries, decode)
		}

		if len(entries) == 0 {
			fmt.Fprintln(os.Stderr, "No matching entries found.")
			return nil
		}

		return extractSingleEntry(cmd, &entries[0], decode)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().Int("index", -1, "Extract the entry at the specified index")
	extractCmd.Flags().Bool("decode", true, "Automatically decode base64/compressed content")
	extractCmd.Flags().Bool("all", false, "Extract all matching entries")
}

// extractSingleEntry extracts response content from a single entry.
func extractSingleEntry(cmd *cobra.Command, entry *har.Entries, decode bool) error {
	if decode {
		data, err := entry.DecodeContent()
		if err != nil {
			return fmt.Errorf("failed to decode content: %w", err)
		}
		if data == nil {
			fmt.Fprintln(os.Stderr, "This entry has no response content.")
			return nil
		}
		return internal.WriteStringOutput(cmd, string(data))
	}

	// Output the raw text without decoding.
	if entry.Response.Content.Text == "" {
		fmt.Fprintln(os.Stderr, "This entry has no response content.")
		return nil
	}
	return internal.WriteStringOutput(cmd, entry.Response.Content.Text)
}

// extractMultipleEntries extracts response content from multiple entries.
func extractMultipleEntries(cmd *cobra.Command, entries []har.Entries, decode bool) error {
	var sb strings.Builder

	for i, entry := range entries {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		sb.WriteString(fmt.Sprintf("# Entry #%d: %s %s\n", i, entry.Request.Method, entry.Request.URL))
		sb.WriteString(fmt.Sprintf("# Status: %d %s\n", entry.Response.Status, entry.Response.StatusText))
		sb.WriteString(fmt.Sprintf("# MIME type: %s\n\n", entry.Response.Content.MimeType))

		if decode {
			data, err := entry.DecodeContent()
			if err != nil {
				sb.WriteString(fmt.Sprintf("# Decode failed: %v\n", err))
				continue
			}
			if data != nil {
				sb.WriteString(string(data))
			} else {
				sb.WriteString("# No content\n")
			}
		} else {
			if entry.Response.Content.Text != "" {
				sb.WriteString(entry.Response.Content.Text)
			} else {
				sb.WriteString("# No content\n")
			}
		}
	}

	return internal.WriteStringOutput(cmd, sb.String())
}
