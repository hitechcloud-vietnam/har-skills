package internal

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// OutputFormat is an output format.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatCSV  OutputFormat = "csv"
	FormatYAML OutputFormat = "yaml"
)

// GetFormat gets the output format from the command-line flag.
func GetFormat(cmd *cobra.Command) OutputFormat {
	f, _ := cmd.Flags().GetString("format")
	switch f {
	case "json":
		return FormatJSON
	case "csv":
		return FormatCSV
	case "yaml":
		return FormatYAML
	default:
		return FormatText
	}
}

// GetOutputPath gets the output file path from the command-line flag.
func GetOutputPath(cmd *cobra.Command) string {
	path, _ := cmd.Flags().GetString("output")
	return path
}

// NoHeader gets whether table headers should be hidden from the command-line flag.
func NoHeader(cmd *cobra.Command) bool {
	nh, _ := cmd.Flags().GetBool("no-header")
	return nh
}

// WriteOutput writes data in the requested format.
// data is the structure to serialize as JSON or YAML.
// textFunc returns text-formatted output.
// csvFunc returns CSV-formatted output.
func WriteOutput(cmd *cobra.Command, data interface{}, textFunc func() string, csvFunc func() string) error {
	format := GetFormat(cmd)
	outputPath := GetOutputPath(cmd)

	var output []byte
	var err error

	switch format {
	case FormatJSON:
		output, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON serialization failed: %w", err)
		}
		output = append(output, '\n')
	case FormatCSV:
		if csvFunc != nil {
			output = []byte(csvFunc())
		} else {
			output, err = json.Marshal(data)
			if err != nil {
				return fmt.Errorf("CSV serialization failed: %w", err)
			}
		}
	case FormatYAML:
		// Use the SDK's YAML functionality.
		if yamlMarshaler, ok := data.(interface{ ToYAML() (string, error) }); ok {
			yamlStr, yamlErr := yamlMarshaler.ToYAML()
			if yamlErr != nil {
				return fmt.Errorf("YAML serialization failed: %w", yamlErr)
			}
			output = []byte(yamlStr)
		} else {
			// Simple fallback: use indented JSON.
			output, err = json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Errorf("serialization failed: %w", err)
			}
			output = append(output, '\n')
		}
	default: // text
		if textFunc != nil {
			output = []byte(textFunc())
		} else {
			output, err = json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Errorf("serialization failed: %w", err)
			}
			output = append(output, '\n')
		}
	}

	return WriteToFileOrStdout(outputPath, output)
}

// WriteStringOutput writes a string to a file or stdout.
func WriteStringOutput(cmd *cobra.Command, content string) error {
	outputPath := GetOutputPath(cmd)
	return WriteToFileOrStdout(outputPath, []byte(content))
}

// WriteToFileOrStdout writes bytes to a file or standard output.
func WriteToFileOrStdout(path string, data []byte) error {
	if path != "" {
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("unable to write file '%s': %w", path, err)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s\n", len(data), path)
		return nil
	}
	_, err := os.Stdout.Write(data)
	return err
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDuration formats milliseconds as a human-readable string.
func FormatDuration(ms float64) string {
	if ms < 1000 {
		return fmt.Sprintf("%.1f ms", ms)
	}
	return fmt.Sprintf("%.2f s", ms/1000)
}
