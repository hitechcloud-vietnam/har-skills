package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// cookieCmd analyzes cookies in a HAR file.
var cookieCmd = &cobra.Command{
	Use:   "cookie",
	Short: "Analyze cookies in a HAR file",
	Long: `Audit cookie security and analyze cookie changes over time.

The security audit checks attributes such as Secure, HttpOnly, and SameSite.
Evolution analysis tracks changes to cookies across requests.

Examples:
  har -f capture.har cookie
  har -f capture.har cookie --audit=false --evolution
  har -f capture.har cookie --name "session_id"
  har -f capture.har cookie --severity medium --format json`,
	RunE: runCookie,
}

func init() {
	rootCmd.AddCommand(cookieCmd)

	cookieCmd.Flags().Bool("audit", true, "Run the cookie security audit")
	cookieCmd.Flags().Bool("evolution", false, "Show the cookie evolution timeline")
	cookieCmd.Flags().String("name", "", "Show only the cookie with the specified name")
	cookieCmd.Flags().String("severity", "info", "Minimum severity filter (info/low/medium/high)")
}

func runCookie(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	_, _ = cmd.Flags().GetBool("audit")
	doEvolution, _ := cmd.Flags().GetBool("evolution")
	cookieName, _ := cmd.Flags().GetString("name")
	severity, _ := cmd.Flags().GetString("severity")

	// Run only the audit unless evolution analysis was requested.
	if !doEvolution {
		report := h.CookieAudit()

		// Filter by severity.
		filtered := filterCookieFindingsBySeverity(report.Findings, severity)
		report.Findings = filtered

		// Filter by cookie name.
		if cookieName != "" {
			var nameFiltered []har.CookieFinding
			for _, f := range report.Findings {
				if f.CookieName == cookieName {
					nameFiltered = append(nameFiltered, f)
				}
			}
			report.Findings = nameFiltered
		}

		return internal.WriteOutput(cmd, report, func() string {
			return formatCookieAuditReport(report)
		}, nil)
	}

	// Analyze cookie evolution.
	evolution := h.CookieEvolution()

	// Filter by name.
	if cookieName != "" {
		if entries, ok := evolution[cookieName]; ok {
			evolution = map[string][]har.CookieEvolutionEntry{
				cookieName: entries,
			}
		} else {
			evolution = map[string][]har.CookieEvolutionEntry{}
		}
	}

	return internal.WriteOutput(cmd, evolution, func() string {
		return formatCookieEvolution(evolution)
	}, nil)
}

// filterCookieFindingsBySeverity filters cookie findings by severity.
func filterCookieFindingsBySeverity(findings []har.CookieFinding, minSeverity string) []har.CookieFinding {
	severityOrder := map[string]int{
		"info":   1,
		"low":    2,
		"medium": 3,
		"high":   4,
	}

	minLevel, ok := severityOrder[strings.ToLower(minSeverity)]
	if !ok {
		return findings
	}

	var result []har.CookieFinding
	for _, f := range findings {
		level, exists := severityOrder[f.Severity]
		if exists && level >= minLevel {
			result = append(result, f)
		}
	}
	return result
}

// formatCookieAuditReport formats the cookie audit report as text.
func formatCookieAuditReport(report *har.CookieAuditReport) string {
	var sb strings.Builder

	sb.WriteString("Cookie Security Audit Report\n")
	sb.WriteString("==================\n")
	sb.WriteString(fmt.Sprintf("Total cookies: %d (unique: %d)\n", report.TotalCookies, report.UniqueCookies))
	sb.WriteString(fmt.Sprintf("Security attributes: Secure=%d, HttpOnly=%d, SameSite=%d\n\n",
		report.SecureCount, report.HttpOnlyCount, report.SameSiteCount))

	if len(report.Findings) == 0 {
		sb.WriteString("No cookie security issues found.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found %d issues:\n", len(report.Findings)))
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString(fmt.Sprintf("%-8s %-15s %-20s %s\n", "Severity", "Category", "Cookie Name", "Description"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for _, f := range report.Findings {
		sb.WriteString(fmt.Sprintf("%-8s %-15s %-20s %s\n",
			strings.ToUpper(f.Severity), f.Category, f.CookieName, f.Description))
	}

	return sb.String()
}

// formatCookieEvolution formats the cookie evolution timeline as text.
func formatCookieEvolution(evolution map[string][]har.CookieEvolutionEntry) string {
	var sb strings.Builder

	sb.WriteString("Cookie Evolution Timeline\n")
	sb.WriteString("================\n\n")

	if len(evolution) == 0 {
		sb.WriteString("No cookie changes found.\n")
		return sb.String()
	}

	for name, entries := range evolution {
		sb.WriteString(fmt.Sprintf("Cookie: %s\n", name))
		sb.WriteString(strings.Repeat("-", 40) + "\n")

		for i, e := range entries {
			sb.WriteString(fmt.Sprintf("  #%d [Entry %d]\n", i+1, e.EntryIndex))
			sb.WriteString(fmt.Sprintf("    Value:   %s\n", truncateString(e.Value, 50)))
			sb.WriteString(fmt.Sprintf("    Secure:  %v  HttpOnly: %v  SameSite: %s\n",
				e.Secure, e.HttpOnly, e.SameSite))
			sb.WriteString(fmt.Sprintf("    Domain:  %s  Path: %s\n", e.Domain, e.Path))
			if i < len(entries)-1 {
				sb.WriteString("    |\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateString truncates strings that exceed the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
