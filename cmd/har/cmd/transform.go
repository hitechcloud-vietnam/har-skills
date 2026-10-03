package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// transformCmd transforms requests in a HAR file.
var transformCmd = &cobra.Command{
	Use:   "transform",
	Short: "Transform requests in a HAR file",
	Long: `Apply transformations to requests in a HAR file, including URL rewrites,
adding or removing headers, changing schemes, and removing query parameters.

Rules are applied in order, and the result is output as HAR JSON.

Examples:
  har -f capture.har transform --rewrite-url "http://localhost->https://prod.example.com"
  har -f capture.har transform --remove-header "X-Debug" --add-header "X-Env:production"
  har -f capture.har transform --change-scheme "http->https" --remove-query-param "_"`,
	RunE: runTransform,
}

func init() {
	rootCmd.AddCommand(transformCmd)

	transformCmd.Flags().StringSlice("rewrite-url", nil, "URL rewrite rule (format: from->to)")
	transformCmd.Flags().StringSlice("remove-header", nil, "Remove the specified request headers")
	transformCmd.Flags().StringSlice("add-header", nil, "Add a request header (format: name:value)")
	transformCmd.Flags().String("add-header-target", "both", "Target for added headers (request/response/both)")
	transformCmd.Flags().StringSlice("change-scheme", nil, "Scheme change rule (format: from->to)")
	transformCmd.Flags().StringSlice("remove-query-param", nil, "Remove the specified query parameters")
	transformCmd.Flags().Bool("in-place", false, "Modify the input file in place")
}

func runTransform(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	var rules []har.TransformRule

	// Parse --rewrite-url rules.
	rewriteURLs, _ := cmd.Flags().GetStringSlice("rewrite-url")
	for _, rw := range rewriteURLs {
		from, to, err := parseArrowRule(rw)
		if err != nil {
			return fmt.Errorf("invalid rewrite-url rule '%s': %w", rw, err)
		}
		rules = append(rules, har.TransformRule{
			Type:        har.TransformURLRewrite,
			Pattern:     from,
			Replacement: to,
		})
	}

	// Parse --remove-header.
	removeHeaders, _ := cmd.Flags().GetStringSlice("remove-header")
	for _, name := range removeHeaders {
		rules = append(rules, har.TransformRule{
			Type:       har.TransformHeaderRemove,
			HeaderName: name,
		})
	}

	// Parse --add-header.
	addHeaders, _ := cmd.Flags().GetStringSlice("add-header")
	addHeaderTarget, _ := cmd.Flags().GetString("add-header-target")
	if len(addHeaders) > 0 {
		headersMap := make(map[string]string)
		for _, h := range addHeaders {
			name, value, err := parseColonKeyValue(h)
			if err != nil {
				return fmt.Errorf("invalid add-header rule '%s': %w", h, err)
			}
			headersMap[name] = value
		}
		// AddHeaders has its own logic, so call it directly.
		result := h.AddHeaders(headersMap, addHeaderTarget)
		h = result
	}

	// Parse --change-scheme.
	changeSchemes, _ := cmd.Flags().GetStringSlice("change-scheme")
	for _, cs := range changeSchemes {
		from, to, err := parseArrowRule(cs)
		if err != nil {
			return fmt.Errorf("invalid change-scheme rule '%s': %w", cs, err)
		}
		rules = append(rules, har.TransformRule{
			Type:        har.TransformSchemeChange,
			Pattern:     from,
			Replacement: to,
		})
	}

	// Parse --remove-query-param.
	removeQueryParams, _ := cmd.Flags().GetStringSlice("remove-query-param")
	for _, param := range removeQueryParams {
		rules = append(rules, har.TransformRule{
			Type:    har.TransformQueryParamRemove,
			Pattern: param,
		})
	}

	// Apply the general transformation rules.
	if len(rules) > 0 {
		inPlace, _ := cmd.Flags().GetBool("in-place")
		if inPlace {
			h.TransformInPlace(rules)
		} else {
			h = h.Transform(rules)
		}
	}

	// Output the transformed HAR JSON.
	data, err := h.ToJSON(true)
	if err != nil {
		return fmt.Errorf("failed to serialize HAR: %w", err)
	}

	return internal.WriteStringOutput(cmd, string(data))
}

// parseArrowRule parses a rule in "from->to" format.
func parseArrowRule(s string) (string, string, error) {
	parts := strings.SplitN(s, "->", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format: from->to")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// parseColonKeyValue parses a key-value pair in "name:value" format.
func parseColonKeyValue(s string) (string, string, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format: name:value")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
