package cmd

import (
	"encoding/json"
	"fmt"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// mergeCmd merges multiple HAR files.
var mergeCmd = &cobra.Command{
	Use:   "merge <file1> [file2] ...",
	Short: "Merge multiple HAR files",
	Long: `Merge entries from multiple HAR files into one HAR file.
The merged file uses the version and creator information from the first file.

Examples:
  har merge capture1.har capture2.har               # Merge two HAR files
  har merge a.har b.har c.har --deduplicate         # Merge and deduplicate
  har merge a.har b.har --sort-by-time=false        # Do not sort by time
  har merge a.har b.har -o merged.har               # Write to a file`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)

	mergeCmd.Flags().Bool("sort-by-time", true, "Sort merged entries by time")
	mergeCmd.Flags().Bool("deduplicate", false, "Deduplicate by method and URL, keeping the latest entry")
}

// runMerge executes the merge command.
func runMerge(cmd *cobra.Command, args []string) error {
	// Load all HAR files.
	hars := make([]*har.Har, 0, len(args))
	for _, path := range args {
		h := internal.LoadHarFromArg(path)
		hars = append(hars, h)
	}

	// Read merge options.
	sortByTime, _ := cmd.Flags().GetBool("sort-by-time")
	deduplicate, _ := cmd.Flags().GetBool("deduplicate")

	options := har.MergeOptions{
		SortByTime:  sortByTime,
		Deduplicate: deduplicate,
	}

	// Merge the files.
	merged := har.MergeWithOptions(options, hars...)

	// Serialize as JSON.
	output, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON serialization failed: %w", err)
	}
	output = append(output, '\n')

	// Write the output.
	outputPath := internal.GetOutputPath(cmd)
	return internal.WriteToFileOrStdout(outputPath, output)
}
