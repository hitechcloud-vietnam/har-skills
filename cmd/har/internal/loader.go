package internal

import (
	"fmt"
	"io"
	"os"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/spf13/cobra"
)

// LoadHar loads a HAR file from the --file flag or stdin.
// If --file is empty and stdin contains data, it reads from stdin.
func LoadHar(cmd *cobra.Command, args []string) *har.Har {
	filePath, _ := cmd.Flags().GetString("file")

	if filePath == "-" || (filePath == "" && hasStdinData()) {
		return LoadHarFromStdin()
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: no HAR file specified. Use -f <file path> or pipe data to stdin.")
		os.Exit(1)
	}

	h, err := LoadHarFromPath(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to load HAR file '%s': %v\n", filePath, err)
		os.Exit(1)
	}
	return h
}

// LoadHarFromPath loads a HAR file from the specified path (with automatic gzip detection).
func LoadHarFromPath(path string) (*har.Har, error) {
	return har.ParseHarFileAuto(path)
}

// LoadHarFromStdin loads HAR data from standard input.
func LoadHarFromStdin() *har.Har {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to read data from stdin: %v\n", err)
		os.Exit(1)
	}

	h, err := har.ParseHar(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to parse HAR data: %v\n", err)
		os.Exit(1)
	}
	return h
}

// LoadHarFromArg loads a HAR file from a command-line argument (for multi-file commands such as diff and merge).
func LoadHarFromArg(path string) *har.Har {
	h, err := LoadHarFromPath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: unable to load HAR file '%s': %v\n", path, err)
		os.Exit(1)
	}
	return h
}

// hasStdinData checks whether stdin has available data.
func hasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
