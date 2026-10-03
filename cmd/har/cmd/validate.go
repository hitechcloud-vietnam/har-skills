package cmd

import (
	"fmt"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// validateCmd checks whether a HAR file conforms to the specification.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a HAR file",
	Long: `Validate a HAR file against the HAR specification.

Standard and strict validation modes are available:
  - Standard validation: check basic structure and required fields
  - Strict validation: additionally check cross-references, HTTP methods, status code ranges, and more
  - Timing consistency validation: compare the Time field with the sum of the Timings fields

Examples:
  har validate -f capture.har                      # Standard validation
  har validate -f capture.har --strict             # Strict validation
  har validate -f capture.har --timings-tolerance 5  # Allow 5 ms timing tolerance
  har validate -f capture.har --strict --timings-tolerance 0`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().Bool("strict", false, "Enable strict validation")
	validateCmd.Flags().Float64("timings-tolerance", 10, "Timing consistency tolerance in milliseconds (0 requires an exact match)")
}

// runValidate executes the validate command.
func runValidate(cmd *cobra.Command, args []string) error {
	// Load the HAR file.
	h := internal.LoadHar(cmd, args)

	strict, _ := cmd.Flags().GetBool("strict")
	timingsTolerance, _ := cmd.Flags().GetFloat64("timings-tolerance")

	// Collect all validation errors.
	var allErrors []*har.ValidationError

	// Run standard or strict validation.
	if strict {
		if err := har.ValidateStrict(h); err != nil {
			collectErrors(err, &allErrors)
		}
	} else {
		if err := har.ValidateHarFile(h); err != nil {
			collectErrors(err, &allErrors)
		}
	}

	// Run timing consistency validation when the tolerance is non-negative.
	if timingsTolerance >= 0 {
		timingErrors := har.ValidateTimingsConsistency(h, timingsTolerance)
		allErrors = append(allErrors, timingErrors...)
	}

	// Write the result in the requested format.
	return internal.WriteOutput(cmd, allErrors,
		func() string { return formatValidateText(allErrors) },
		nil,
	)
}

// collectErrors extracts ValidationErrors from a HarError.
func collectErrors(err error, errors *[]*har.ValidationError) {
	if err == nil {
		return
	}

	// Try to handle the error as a HarError.
	if harErr, ok := err.(*har.HarError); ok {
		for _, pe := range harErr.GetPartialErrors() {
			// Convert the HarError's Field and Message to a ValidationError.
			ve := &har.ValidationError{
				Field:   pe.Field,
				Message: pe.Message,
			}
			*errors = append(*errors, ve)
		}
		return
	}

	// Handle other error types.
	*errors = append(*errors, &har.ValidationError{
		Field:   "",
		Message: err.Error(),
	})
}

// formatValidateText formats validation results as text.
func formatValidateText(errors []*har.ValidationError) string {
	if len(errors) == 0 {
		return "✓ Valid\n"
	}

	result := fmt.Sprintf("✗ Found %d validation error(s):\n\n", len(errors))
	for i, e := range errors {
		if e.Field != "" {
			result += fmt.Sprintf("  %d. [%s] %s: %s\n", i+1, e.Rule, e.Field, e.Message)
		} else {
			result += fmt.Sprintf("  %d. %s\n", i+1, e.Message)
		}
	}
	return result
}
