package cmd

import (
	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// diffCmd compares two HAR files.
var diffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Compare two HAR files",
	Long: `Compare two HAR files to find added, removed, and modified requests.

Examples:
  har diff capture1.har capture2.har              # Compare two HAR files
  har diff a.har b.har --ignore-headers=Cookie    # Ignore differences in the Cookie header
  har diff a.har b.har --compare-by-url           # Match by URL instead of index
  har diff a.har b.har --include-body             # Compare response bodies`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().StringSlice("ignore-headers", nil, "Header names to ignore (comma-separated)")
	diffCmd.Flags().Bool("ignore-timings", true, "Ignore timing differences")
	diffCmd.Flags().Bool("ignore-dates", true, "Ignore date differences")
	diffCmd.Flags().Bool("include-body", false, "Compare response bodies")
	diffCmd.Flags().Bool("compare-by-url", false, "Match by URL (default: index and URL)")
}

// runDiff executes the diff command.
func runDiff(cmd *cobra.Command, args []string) error {
	// Load both HAR files.
	har1 := internal.LoadHarFromArg(args[0])
	har2 := internal.LoadHarFromArg(args[1])

	// Build diff options.
	options := har.DefaultDiffOptions()

	// Read options from command-line flags.
	ignoreHeaders, _ := cmd.Flags().GetStringSlice("ignore-headers")
	ignoreTimings, _ := cmd.Flags().GetBool("ignore-timings")
	ignoreDates, _ := cmd.Flags().GetBool("ignore-dates")
	includeBody, _ := cmd.Flags().GetBool("include-body")
	compareByURL, _ := cmd.Flags().GetBool("compare-by-url")

	options.IgnoreHeaders = ignoreHeaders
	options.IgnoreTimings = ignoreTimings
	options.IgnoreDates = ignoreDates
	options.IncludeBody = includeBody
	options.CompareByURL = compareByURL

	// Compare the files.
	diffResult := har.Diff(har1, har2, options)

	// Write the result in the requested format.
	return internal.WriteOutput(cmd, diffResult,
		func() string { return diffResult.Report(har.FormatText) },
		func() string { return diffResult.Report(har.FormatCSV) },
	)
}
