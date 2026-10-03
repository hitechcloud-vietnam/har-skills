package cmd

import (
	"fmt"
	"strings"

	har "github.com/hitechcloud-vietnam/har-skills"
	"github.com/hitechcloud-vietnam/har-skills/cmd/har/internal"
	"github.com/spf13/cobra"
)

// performanceCmd scores the performance of a HAR file.
var performanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "Score HAR file performance",
	Long: `Score the performance of requests in a HAR file and generate a grade and optimization recommendations.

Scoring categories include TTFB (time to first byte), total load time, request count,
transfer size, cache efficiency, and compression ratio.

Examples:
  har -f capture.har performance
  har -f capture.har performance --format json`,
	RunE: runPerformance,
}

func init() {
	rootCmd.AddCommand(performanceCmd)
}

func runPerformance(cmd *cobra.Command, args []string) error {
	h := internal.LoadHar(cmd, args)

	report := h.PerformanceScore()

	return internal.WriteOutput(cmd, report, func() string {
		return formatPerformanceReport(report)
	}, nil)
}

// formatPerformanceReport formats the performance score report as text.
func formatPerformanceReport(report *har.PerformanceReport) string {
	var sb strings.Builder

	sb.WriteString("Performance Score Report\n")
	sb.WriteString("============\n")

	// Overall score and grade.
	sb.WriteString(fmt.Sprintf("Overall score: %.1f/100  Grade: %s\n\n", report.OverallScore, report.Grade()))

	// Category scores.
	sb.WriteString("Category Scores:\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")
	sb.WriteString(fmt.Sprintf("%-20s %-10s %-10s %s\n", "Category", "Score", "Weight", "Status"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, cat := range report.Categories {
		status := "Excellent"
		if cat.Score < 50 {
			status = "Poor"
		} else if cat.Score < 70 {
			status = "Fair"
		} else if cat.Score < 90 {
			status = "Good"
		}
		sb.WriteString(fmt.Sprintf("%-20s %-10.1f %-10.1f %s\n",
			cat.Name, cat.Score, cat.Weight, status))
	}

	// Findings.
	for _, cat := range report.Categories {
		if len(cat.Findings) > 0 {
			sb.WriteString(fmt.Sprintf("\n%s Details:\n", cat.Name))
			for _, f := range cat.Findings {
				sb.WriteString(fmt.Sprintf("  - [%s] %s\n", f.Type, f.Title))
				if f.Description != "" {
					sb.WriteString(fmt.Sprintf("    %s\n", f.Description))
				}
				if f.Impact != "" {
					sb.WriteString(fmt.Sprintf("    Impact: %s\n", f.Impact))
				}
			}
		}
	}

	// Recommendations.
	if len(report.Recommendations) > 0 {
		sb.WriteString("\nRecommendations:\n")
		sb.WriteString(strings.Repeat("-", 60) + "\n")
		for i, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
	}

	return sb.String()
}
