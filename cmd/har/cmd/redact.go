package cmd

import (
	"encoding/json"
	"fmt"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// redactCmd redacts sensitive data in a HAR file.
var redactCmd = &cobra.Command{
	Use:   "redact",
	Short: "Redact sensitive data",
	Long: `Replace sensitive data in a HAR file, such as passwords, tokens, and API keys, with placeholders.

Custom redaction targets are supported:
  - Header names (such as Authorization and X-Api-Key)
  - Cookie names (such as session and token)
  - Query parameter names (such as password and api_key)
  - POST field names (such as password and secret)
  - IP address anonymization (replace the last segment with 0)

Examples:
  har redact -f capture.har                             # Use the default redaction rules
  har redact -f capture.har --defaults=false            # Do not use the default rules
  har redact -f capture.har --header=X-Custom-Key      # Add a custom header
  har redact -f capture.har --redact-ips                # Anonymize IP addresses
  har redact -f capture.har --replacement="***"          # Set a custom replacement string
  har redact -f capture.har --in-place                  # Modify the file in place`,
	RunE: runRedact,
}

func init() {
	rootCmd.AddCommand(redactCmd)

	redactCmd.Flags().Bool("defaults", true, "Use the default redaction rules")
	redactCmd.Flags().StringSlice("header", nil, "Additional header names to redact")
	redactCmd.Flags().StringSlice("cookie", nil, "Additional cookie names to redact")
	redactCmd.Flags().StringSlice("query-param", nil, "Additional query parameter names to redact")
	redactCmd.Flags().StringSlice("post-field", nil, "Additional POST field names to redact")
	redactCmd.Flags().String("replacement", "[REDACTED]", "Replacement text")
	redactCmd.Flags().Bool("redact-ips", false, "Anonymize IP addresses")
	redactCmd.Flags().Bool("in-place", false, "Modify the file in place")
}

// runRedact executes the redact command.
func runRedact(cmd *cobra.Command, args []string) error {
	// Load the HAR file.
	h := internal.LoadHar(cmd, args)

	// Build redaction options.
	var opts har.RedactOptions

	useDefaults, _ := cmd.Flags().GetBool("defaults")
	if useDefaults {
		opts = har.DefaultRedactOptions()
	} else {
		opts = har.RedactOptions{
			Replacement: "[REDACTED]",
		}
	}

	// Read command-line flags.
	extraHeaders, _ := cmd.Flags().GetStringSlice("header")
	extraCookies, _ := cmd.Flags().GetStringSlice("cookie")
	extraQueryParams, _ := cmd.Flags().GetStringSlice("query-param")
	extraPostFields, _ := cmd.Flags().GetStringSlice("post-field")
	replacement, _ := cmd.Flags().GetString("replacement")
	redactIPs, _ := cmd.Flags().GetBool("redact-ips")
	inPlace, _ := cmd.Flags().GetBool("in-place")

	// Merge additional redaction targets.
	opts.Headers = append(opts.Headers, extraHeaders...)
	opts.Cookies = append(opts.Cookies, extraCookies...)
	opts.QueryParams = append(opts.QueryParams, extraQueryParams...)
	opts.PostDataFields = append(opts.PostDataFields, extraPostFields...)
	opts.Replacement = replacement
	opts.RedactIPs = redactIPs

	// Redact the data.
	var result *har.Har
	if inPlace {
		h.RedactInPlace(opts)
		result = h
	} else {
		result = h.Redact(opts)
	}

	// Serialize as JSON.
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON serialization failed: %w", err)
	}
	output = append(output, '\n')

	// Write the output.
	outputPath := internal.GetOutputPath(cmd)
	if inPlace && outputPath == "" {
		// In-place mode writes back to the path provided with --file.
		filePath, _ := cmd.Flags().GetString("file")
		if filePath != "" && filePath != "-" {
			outputPath = filePath
		}
	}

	return internal.WriteToFileOrStdout(outputPath, output)
}
