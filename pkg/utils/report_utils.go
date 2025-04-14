package utils

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

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
	categorizedChecks := make(map[types.Category][]types.Check)
	for _, check := range checks {
		// Map old categories to new ones for consistent reporting
		category := check.Category()
		switch category {
		case types.CategoryCluster:
			category = types.CategoryClusterConfig
		case types.CategoryNetworking:
			category = types.CategoryNetwork
		case types.CategoryApplications:
			category = types.CategoryAppDev
		case types.CategoryMonitoring:
			category = types.CategoryOpReady
		case types.CategoryInfrastructure:
			category = types.CategoryInfra
		}

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

// OrganizeChecksByCategory groups checks by their category
func OrganizeChecksByCategory(checks []types.Check) map[types.Category][]types.Check {
	categorized := make(map[types.Category][]types.Check)

	for _, check := range checks {
		// Map old categories to new ones for consistent reporting
		category := check.Category()
		switch category {
		case types.CategoryCluster:
			category = types.CategoryClusterConfig
		case types.CategoryNetworking:
			category = types.CategoryNetwork
		case types.CategoryApplications:
			category = types.CategoryAppDev
		case types.CategoryMonitoring:
			category = types.CategoryOpReady
		case types.CategoryInfrastructure:
			category = types.CategoryInfra
		}

		categorized[category] = append(categorized[category], check)
	}

	return categorized
}

// GetSortedCategories returns categories in the preferred order
func GetSortedCategories() []types.Category {
	return []types.Category{
		types.CategoryInfra,
		types.CategoryNetwork,
		types.CategoryStorage,
		types.CategoryClusterConfig,
		types.CategoryAppDev,
		types.CategorySecurity,
		types.CategoryOpReady,
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
		sb.WriteString("[source, bash]\n----\n")
		sb.WriteString(result.Detail)
		sb.WriteString("\n----\n\n")
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

// GetResultByStatus returns a map of results grouped by status
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
