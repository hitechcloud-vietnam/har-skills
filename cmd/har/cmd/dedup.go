package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// dedupCmd finds or removes duplicate requests.
var dedupCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Find or remove duplicate requests",
	Long: `Find duplicate or near-duplicate requests in a HAR file, or remove duplicates to create a new HAR file.

Three deduplication strategies are supported:
  - exact:        Match exact URLs
  - pattern:      Match URL patterns while ignoring cache-busting parameters (default)
  - content-hash: Match by content hash

Examples:
  har -f capture.har dedup
  har -f capture.har dedup --strategy exact
  har -f capture.har dedup --strategy content-hash --compare-headers --compare-body
  har -f capture.har dedup --remove -o cleaned.har
  har -f capture.har dedup --ignore-param "timestamp" --ignore-param "_"`,
	RunE: runDedup,
}

func init() {
	rootCmd.AddCommand(dedupCmd)

	dedupCmd.Flags().String("strategy", "pattern", "Deduplication strategy (exact/pattern/content-hash)")
	dedupCmd.Flags().StringSlice("ignore-param", nil, "Query parameters to ignore during deduplication")
	dedupCmd.Flags().Bool("compare-headers", false, "Include request headers when comparing")
	dedupCmd.Flags().Bool("compare-body", false, "Include request bodies when comparing")
	dedupCmd.Flags().Bool("remove", false, "Remove duplicate requests and output a cleaned HAR file")
}

func runDedup(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	// Build deduplication options.
	strategyName, _ := cmd.Flags().GetString("strategy")
	ignoreParams, _ := cmd.Flags().GetStringSlice("ignore-param")
	compareHeaders, _ := cmd.Flags().GetBool("compare-headers")
	compareBody, _ := cmd.Flags().GetBool("compare-body")
	doRemove, _ := cmd.Flags().GetBool("remove")

	// Parse the strategy.
	strategy := parseDedupStrategy(strategyName)

	// Start with default options and override the specified fields.
	opts := har.DefaultDeduplicateOptions()
	opts.Strategy = strategy
	opts.CompareHeaders = compareHeaders
	opts.CompareBody = compareBody

	if len(ignoreParams) > 0 {
		opts.IgnoreParams = ignoreParams
	}

	if doRemove {
		// Remove duplicates and output the cleaned HAR file.
		deduped := h.Deduplicate(opts)
		data, err := deduped.ToJSON(true)
		if err != nil {
			return fmt.Errorf("failed to serialize HAR: %w", err)
		}
		return internal.WriteStringOutput(cmd, string(data))
	}

	// Find duplicates by default.
	groups := h.FindDuplicates(opts)

	return internal.WriteOutput(cmd, groups, func() string {
		return formatDuplicateGroups(groups)
	}, nil)
}

// parseDedupStrategy parses a deduplication strategy string.
func parseDedupStrategy(s string) har.DedupStrategy {
	switch strings.ToLower(s) {
	case "exact":
		return har.DedupExactURL
	case "pattern":
		return har.DedupURLPattern
	case "content-hash":
		return har.DedupContentHash
	default:
		return har.DedupURLPattern
	}
}

// formatDuplicateGroups formats duplicate groups as text.
func formatDuplicateGroups(groups []har.DuplicateGroup) string {
	if len(groups) == 0 {
		return "No duplicate requests found.\n"
	}

	var sb strings.Builder
	sb.WriteString("Duplicate Request Analysis\n")
	sb.WriteString("============\n\n")

	totalDuplicates := 0
	for _, g := range groups {
		totalDuplicates += g.Count - 1 // The first entry is not considered a duplicate.
	}

	sb.WriteString(fmt.Sprintf("Found %d groups of duplicate requests, totaling %d duplicate entries\n\n",
		len(groups), totalDuplicates))

	sb.WriteString(fmt.Sprintf("%-60s %-6s %s\n", "Deduplication Key", "Count", "Entry Indices"))
	sb.WriteString(strings.Repeat("-", 90) + "\n")

	for _, g := range groups {
		keyDisplay := g.Key
		if len(keyDisplay) > 60 {
			keyDisplay = keyDisplay[:57] + "..."
		}
		indices := fmt.Sprintf("%v", g.EntryIndices)
		if len(indices) > 30 {
			indices = indices[:27] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-60s %-6d %s\n", keyDisplay, g.Count, indices))
	}

	return sb.String()
}
