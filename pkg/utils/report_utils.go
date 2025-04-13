package utils

import (
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// GenerateFullAsciiDocReport generates a complete AsciiDoc report for all health checks
func GenerateFullAsciiDocReport(title string, checks []types.Check, results map[string]types.Result) string {
	var sb strings.Builder

	// Get OpenShift version for documentation links
	version, err := GetOpenShiftMajorMinorVersion()
	if err != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Generate report header
	sb.WriteString(GenerateAsciiDocReportHeader(title))

	// Organize checks by category
	categorizedChecks := make(map[types.Category][]types.Check)
	for _, check := range checks {
		category := check.Category()
		categorizedChecks[category] = append(categorizedChecks[category], check)
	}

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

		sb.WriteString(GenerateAsciiDocCheckSection(check, result, version))
		sb.WriteString("\n\n")
	}

	// Reset bgcolor for future tables
	sb.WriteString("// Reset bgcolor for future tables\n[grid=none,frame=none]\n|===\n|{set:cellbgcolor!}\n|===\n\n")

	return sb.String()
}
