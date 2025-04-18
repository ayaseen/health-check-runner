/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file provides utilities for generating various report formats. It includes:

- Functions for creating full-featured AsciiDoc reports
- Methods for organizing checks by category
- Utilities for formatting detailed check information
- Functions for generating issue summaries
- Helpers for consistent report structure across different formats

These utilities support the creation of comprehensive, well-organized health check reports that highlight issues and recommendations.
*/

package utils

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// IsAlreadyFormatted checks if text already contains source blocks or other formatting
func IsAlreadyFormatted(text string) bool {
	// Check for AsciiDoc source blocks with different variations
	if strings.Contains(text, "[source,") ||
		strings.Contains(text, "[source, ") ||
		strings.Contains(text, "[source=") ||
		strings.Contains(text, "----") ||
		strings.Contains(text, "....") {
		return true
	}

	// Check for AsciiDoc tables
	if strings.Contains(text, "|===") ||
		strings.Contains(text, "[cols=") {
		return true
	}

	// Check for other AsciiDoc formatting elements
	if strings.Contains(text, "== ") ||
		strings.Contains(text, "=== ") {
		return true
	}

	// Check for complex YAML or JSON structure
	if (strings.Contains(text, "apiVersion:") && strings.Contains(text, "kind:")) ||
		(strings.Contains(text, "metadata:") && strings.Contains(text, "spec:")) {
		return true
	}

	return false
}

// FormatAsCodeBlock formats text as a source code block with the appropriate language
// Ensures proper AsciiDoc format with correct delimiter spacing
func FormatAsCodeBlock(content string, language string) string {
	// Skip if already formatted
	if IsAlreadyFormatted(content) {
		return content
	}

	// Default to yaml for structured data with common patterns
	if language == "" {
		if (strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")) ||
			(strings.Contains(content, "metadata:") && strings.Contains(content, "spec:")) {
			language = "yaml"
		} else if strings.HasPrefix(strings.TrimSpace(content), "{") && strings.Contains(content, "\":") {
			language = "json"
		} else if strings.Contains(content, "NAME") && strings.Contains(content, "READY") {
			language = "bash"
		} else {
			language = "text"
		}
	}

	// Ensure proper spacing in the AsciiDoc format
	// This is the critical fix - ensuring the exact format expected by AsciiDoc
	return fmt.Sprintf("[source, %s]\n----\n%s\n----\n", language, content)
}

// GenerateFullAsciiDocReport generates a complete AsciiDoc report for all health checks
func GenerateFullAsciiDocReport(title string, checks []types.Check, results map[string]types.Result) string {
	var sb strings.Builder

	// Get OpenShift version for documentation links
	version, err := GetOpenShiftMajorMinorVersion()
	if err != nil {
		version = "4.14" // Default to a known version if we can't determine
	}

	// Generate report header
	sb.WriteString(GenerateAsciiDocReportHeader(title))

	// Organize checks by category
	categorizedChecks := OrganizeChecksByCategory(checks)

	// Generate summary table
	sb.WriteString(GenerateAsciiDocSummaryTable(checks, results))

	// Generate category sections
	sb.WriteString(GenerateAsciiDocCategorySections(categorizedChecks, results))

	// Generate detailed sections for each check
	sb.WriteString("<<<\n\n{set:cellbgcolor!}\n\n")
	for _, check := range checks {
		result, exists := results[check.ID()]
		if !exists {
			continue
		}

		sb.WriteString(FormatCheckDetail(check, result, version))
		sb.WriteString("\n\n")
	}

	// Reset bgcolor for future tables
	sb.WriteString("// Reset bgcolor for future tables\n[grid=none,frame=none]\n|===\n|{set:cellbgcolor!}\n|===\n\n")

	return sb.String()
}

// OrganizeChecksByCategory groups checks by their category
func OrganizeChecksByCategory(checks []types.Check) map[types.Category][]types.Check {
	categorized := make(map[types.Category][]types.Check)

	for _, check := range checks {
		category := check.Category()
		categorized[category] = append(categorized[category], check)
	}

	return categorized
}

// GetSortedCategories returns categories in the preferred order
func GetSortedCategories() []types.Category {
	return []types.Category{
		types.CategoryClusterConfig,
		types.CategoryNetworking,
		types.CategoryApplications,
		types.CategoryOpReady,
		types.CategorySecurity,
		types.CategoryStorage,
		types.CategoryPerformance,
	}
}

// FormatCheckDetail formats detailed information about a check
func FormatCheckDetail(check types.Check, result types.Result, version string) string {
	var sb strings.Builder

	// Add section with check name
	sb.WriteString(fmt.Sprintf("== %s\n\n", check.Name()))

	// Add result status
	sb.WriteString(GetChanges(result.ResultKey) + "\n\n")

	// Add detail if available
	if result.Detail != "" {
		// Check if the detail already contains source blocks or formatted content
		if IsAlreadyFormatted(result.Detail) {
			// If content is already formatted, include it as is
			sb.WriteString(result.Detail)

			// Ensure there's proper spacing after the detail
			if !strings.HasSuffix(result.Detail, "\n\n") {
				if strings.HasSuffix(result.Detail, "\n") {
					sb.WriteString("\n")
				} else {
					sb.WriteString("\n\n")
				}
			}
		} else {
			// Identify appropriate language based on content pattern and format it
			language := ""
			sb.WriteString(FormatAsCodeBlock(result.Detail, language))
		}
	}

	// Add observation
	sb.WriteString("**Observation**\n\n")
	sb.WriteString(result.Message + "\n\n")

	// Add recommendations
	sb.WriteString("**Recommendation**\n\n")
	if len(result.Recommendations) > 0 {
		for _, rec := range result.Recommendations {
			sb.WriteString(rec + "\n\n")
		}
	} else {
		sb.WriteString("None\n\n")
	}

	// Add reference links
	sb.WriteString("*Reference Link(s)*\n\n")
	sb.WriteString(fmt.Sprintf("* https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/\n\n", version))

	return sb.String()
}

// GetResultsByStatus returns a map of results grouped by status
func GetResultsByStatus(results map[string]types.Result) map[types.Status][]types.Result {
	resultsByStatus := make(map[types.Status][]types.Result)

	for _, result := range results {
		resultsByStatus[types.Status(result.Status)] = append(resultsByStatus[types.Status(result.Status)], result)
	}

	return resultsByStatus
}

// CountResultsByStatus counts the number of results in each status
func CountResultsByStatus(results map[string]types.Result) map[types.Status]int {
	counts := make(map[types.Status]int)

	for _, result := range results {
		counts[types.Status(result.Status)]++
	}

	return counts
}

// FormatIssuesSummary formats a summary of issues (warnings and critical findings)
func FormatIssuesSummary(checks []types.Check, results map[string]types.Result) string {
	var sb strings.Builder

	sb.WriteString("Issues found:\n\n")

	resultsByStatus := GetResultsByStatus(results)

	// First show critical issues
	if criticalResults, ok := resultsByStatus[types.StatusCritical]; ok && len(criticalResults) > 0 {
		sb.WriteString("Critical issues:\n")
		for _, result := range criticalResults {
			// Find the check name
			var checkName string
			for _, check := range checks {
				if check.ID() == result.CheckID {
					checkName = check.Name()
					break
				}
			}

			sb.WriteString(fmt.Sprintf("- [Critical] %s: %s\n", checkName, result.Message))

			if len(result.Recommendations) > 0 {
				sb.WriteString("  Recommendations:\n")
				for _, rec := range result.Recommendations {
					sb.WriteString(fmt.Sprintf("  - %s\n", rec))
				}
				sb.WriteString("\n")
			}
		}
	}

	// Then show warnings
	if warningResults, ok := resultsByStatus[types.StatusWarning]; ok && len(warningResults) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, result := range warningResults {
			// Find the check name
			var checkName string
			for _, check := range checks {
				if check.ID() == result.CheckID {
					checkName = check.Name()
					break
				}
			}

			sb.WriteString(fmt.Sprintf("- [Warning] %s: %s\n", checkName, result.Message))

			if len(result.Recommendations) > 0 {
				sb.WriteString("  Recommendations:\n")
				for _, rec := range result.Recommendations {
					sb.WriteString(fmt.Sprintf("  - %s\n", rec))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
