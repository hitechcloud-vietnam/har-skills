package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// securityCmd runs a security audit.
var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Run a security audit on a HAR file",
	Long: `Audit requests and responses in a HAR file for the following issues:

  - Missing security headers (Strict-Transport-Security, X-Content-Type-Options, etc.)
  - Cookie security (Secure, HttpOnly, and SameSite attributes)
  - Mixed content (HTTP resources on HTTPS pages)
  - Sensitive data exposure (API keys, tokens, etc.)
  - CORS configuration issues
  - Information disclosure (Server headers, error messages, etc.)

Examples:
  har -f capture.har security
  har -f capture.har security --severity high
  har -f capture.har security --check-cookies=false --format json`,
	RunE: runSecurity,
}

func init() {
	rootCmd.AddCommand(securityCmd)

	securityCmd.Flags().Bool("check-headers", true, "Check security headers")
	securityCmd.Flags().Bool("check-cookies", true, "Check cookie security")
	securityCmd.Flags().Bool("check-mixed-content", true, "Check for mixed content")
	securityCmd.Flags().Bool("check-sensitive-data", true, "Check for sensitive data exposure")
	securityCmd.Flags().Bool("check-cors", true, "Check CORS configuration")
	securityCmd.Flags().Bool("check-info-disclosure", true, "Check for information disclosure")
	securityCmd.Flags().String("severity", "low", "Minimum severity filter (all/info/low/medium/high)")
}

func runSecurity(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	// Build audit options.
	opts := har.SecurityAuditOptions{
		CheckSecurityHeaders: mustGetBool(cmd, "check-headers"),
		CheckCookies:         mustGetBool(cmd, "check-cookies"),
		CheckMixedContent:    mustGetBool(cmd, "check-mixed-content"),
		CheckSensitiveData:   mustGetBool(cmd, "check-sensitive-data"),
		CheckCORS:            mustGetBool(cmd, "check-cors"),
		CheckInfoDisclosure:  mustGetBool(cmd, "check-info-disclosure"),
	}

	report := h.SecurityAuditWithOptions(opts)

	// Filter by severity.
	severity, _ := cmd.Flags().GetString("severity")
	filtered := filterFindingsBySeverity(report.Findings, severity)
	report.Findings = filtered

	return internal.WriteOutput(cmd, report, func() string {
		return formatSecurityReport(report, severity)
	}, nil)
}

// filterFindingsBySeverity filters findings at or above the minimum severity.
func filterFindingsBySeverity(findings []har.SecurityFinding, minSeverity string) []har.SecurityFinding {
	severityOrder := map[string]int{
		"all":    0,
		"info":   1,
		"low":    2,
		"medium": 3,
		"high":   4,
	}

	minLevel, ok := severityOrder[strings.ToLower(minSeverity)]
	if !ok || minSeverity == "all" {
		return findings
	}

	var result []har.SecurityFinding
	for _, f := range findings {
		level, exists := severityOrder[f.Severity]
		if exists && level >= minLevel {
			result = append(result, f)
		}
	}
	return result
}

// formatSecurityReport formats the security audit report as text.
func formatSecurityReport(report *har.SecurityReport, severity string) string {
	var sb strings.Builder

	sb.WriteString("Security Audit Report\n")
	sb.WriteString("============\n")
	sb.WriteString(fmt.Sprintf("Score: %d/100\n", report.Score))
	sb.WriteString(fmt.Sprintf("Findings: %d issue(s)\n\n", len(report.Findings)))

	if len(report.Findings) == 0 {
		sb.WriteString("No security issues found.\n")
		return sb.String()
	}

	// Group by severity.
	groups := map[string][]har.SecurityFinding{
		"high":   {},
		"medium": {},
		"low":    {},
		"info":   {},
	}
	for _, f := range report.Findings {
		groups[f.Severity] = append(groups[f.Severity], f)
	}

	severityLabels := []string{"high", "medium", "low", "info"}
	severityNames := map[string]string{
		"high":   "High",
		"medium": "Medium",
		"low":    "Low",
		"info":   "Info",
	}

	for _, sev := range severityLabels {
		findings := groups[sev]
		if len(findings) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("[%s] %s (%d)\n", strings.ToUpper(sev), severityNames[sev], len(findings)))
		sb.WriteString(strings.Repeat("-", 60) + "\n")

		for i, f := range findings {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, f.Title))
			if f.EntryURL != "" {
				sb.WriteString(fmt.Sprintf("     URL: %s\n", f.EntryURL))
			}
			sb.WriteString(fmt.Sprintf("     Category: %s\n", f.Category))
			sb.WriteString(fmt.Sprintf("     Description: %s\n", f.Description))
			if f.Remedy != "" {
				sb.WriteString(fmt.Sprintf("     Remediation: %s\n", f.Remedy))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// mustGetBool reads a boolean command-line flag.
func mustGetBool(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}
