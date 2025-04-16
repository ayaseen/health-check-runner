/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file provides enhanced AsciiDoc report generation utilities. It includes:

- Functions for creating comprehensive and well-formatted AsciiDoc reports
- Methods for organizing health check results by category
- Utilities for formatting result keys and status indicators
- Functions for creating detailed check sections with recommendations
- Helpers for generating summary tables and detailed explanations

These utilities enhance the standard AsciiDoc generation with more sophisticated formatting and organization.
*/

package utils

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// GenerateEnhancedAsciiDocReport generates a comprehensive AsciiDoc report that matches the old report format
func GenerateEnhancedAsciiDocReport(title string, checks []types.Check, results map[string]types.Result) string {
	var sb strings.Builder

	// Add report header with title
	sb.WriteString(fmt.Sprintf("= %s\n\n", title))

	// Add AsciiDoc settings for GitHub
	sb.WriteString("ifdef::env-github[]\n:tip-caption: :bulb:\n:note-caption: :information_source:\n:important-caption: :heavy_exclamation_mark:\n:caution-caption: :fire:\n:warning-caption: :warning:\nendif::[]\n\n")

	// Add key section with color coding and descriptions - exactly matching old format
	sb.WriteString("= Key\n\n")
	sb.WriteString("[cols=\"1,3\", options=header]\n|===\n|Value\n|Description\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FF0000}\nChanges Required\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Indicates Changes Required for system stability, subscription compliance, or other reason.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FEFE20}\nChanges Recommended\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Indicates Changes Recommended to align with recommended practices, but not urgently required\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#A6B9BF}\nN/A\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No advise given on line item.  For line items which are data-only to provide context.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#80E5FF}\nAdvisory\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No change required or recommended, but additional information provided.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#00FF00}\nNo Change\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No change required. In alignment with recommended practices.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FFFFFF}\nTo Be Evaluated\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Not yet evaluated. Will appear only in draft copies.\n|===\n\n")

	// Generate summary section with all checks - matching old format
	sb.WriteString("= Summary\n\n")
	sb.WriteString("\n[cols=\"1,2,2,3\", options=header]\n|===\n|*Category*\n|*Item Evaluated*\n|*Observed Result*\n|*Recommendation*\n\n")

	// Group checks by category for the summary
	checksByCategory := groupChecksByCategory(checks, results)

	// Add all checks to the summary section
	// Order categories as in the old report
	orderedCategories := []types.Category{
		types.CategoryInfra,
		types.CategoryNetwork,
		types.CategoryStorage,
		types.CategoryClusterConfig,
		types.CategoryAppDev,
		types.CategorySecurity,
		types.CategoryOpReady,
	}

	for _, category := range orderedCategories {
		categoryChecks, exists := checksByCategory[category]
		if !exists {
			continue
		}

		for _, check := range categoryChecks {
			result, exists := results[check.ID()]
			if !exists {
				continue
			}

			// Category column
			sb.WriteString("// ------------------------ITEM START\n")
			sb.WriteString("// ----ITEM SOURCE:  ./content/healthcheck-items/" + check.ID() + ".item\n\n")
			sb.WriteString("// Category\n")
			sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

			// Item Evaluated column with link to detailed section
			sb.WriteString("// Item Evaluated\n")
			sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

			// Observed Result column
			sb.WriteString("| " + result.Message + " \n\n")

			// Recommendation column with proper coloring
			sb.WriteString(formatResultKeyForSummaryTable(result.ResultKey) + "\n\n")
			sb.WriteString("// ------------------------ITEM END\n\n")
		}
	}

	sb.WriteString("|===\n\n")
	sb.WriteString("<<<\n\n")
	sb.WriteString("{set:cellbgcolor!}\n\n")

	// Add detailed category sections
	for _, category := range orderedCategories {
		categoryChecks, exists := checksByCategory[category]
		if !exists {
			continue
		}

		// Add category heading with proper formatting
		sb.WriteString("# " + string(category) + "\n\n")

		// Start category table
		sb.WriteString("[cols=\"1,2,2,3\", options=header]\n|===\n|*Category*\n|*Item Evaluated*\n|*Observed Result*\n|*Recommendation*\n\n")

		// Add all checks for this category
		for _, check := range categoryChecks {
			result, exists := results[check.ID()]
			if !exists {
				continue
			}

			// Category column
			sb.WriteString("// ------------------------ITEM START\n")
			sb.WriteString("// ----ITEM SOURCE:  ./content/healthcheck-items/" + check.ID() + ".item\n\n")
			sb.WriteString("// Category\n")
			sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

			// Item Evaluated column with link to detailed section
			sb.WriteString("// Item Evaluated\n")
			sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

			// Observed Result column
			sb.WriteString("| " + result.Message + " \n\n")

			// Recommendation column with proper coloring
			sb.WriteString(formatResultKeyForSummaryTable(result.ResultKey) + "\n\n")
			sb.WriteString("// ------------------------ITEM END\n")
		}

		sb.WriteString("|===\n\n")
		sb.WriteString("<<<\n\n")
		sb.WriteString("{set:cellbgcolor!}\n\n")
	}

	// Get OpenShift version for documentation links
	version, err := GetOpenShiftMajorMinorVersion()
	if err != nil {
		version = "4.14" // Default to a known version if we can't determine
	}

	// Add detailed sections for each check
	for _, category := range orderedCategories {
		categoryChecks, exists := checksByCategory[category]
		if !exists {
			continue
		}

		for _, check := range categoryChecks {
			result, exists := results[check.ID()]
			if !exists {
				continue
			}

			// Add section header with check name
			sb.WriteString("== " + check.Name() + "\n\n")

			// Add result key formatted as a centered table (exactly like old report)
			sb.WriteString(formatResultKeyAsCenteredTable(result.ResultKey) + "\n\n")

			// Add detailed output if available, properly formatted
			if result.Detail != "" {
				if containsAsciiDocFormatting(result.Detail) {
					// If result.Detail already contains AsciiDoc formatting, include it directly
					sb.WriteString(result.Detail)
					sb.WriteString("\n\n")
				} else {
					// Otherwise, wrap it in a source block
					sb.WriteString("[source, bash]\n----\n")
					sb.WriteString(result.Detail)
					sb.WriteString("\n----\n\n")
				}
			}

			// Add observation section
			sb.WriteString("**Observation**\n\n")
			sb.WriteString(result.Message + "\n\n")

			// Add recommendation section
			sb.WriteString("**Recommendation**\n\n")
			if len(result.Recommendations) > 0 {
				for _, rec := range result.Recommendations {
					sb.WriteString(rec + "\n\n")
				}
			} else {
				sb.WriteString("None\n\n")
			}

			// Add reference links section (similar to old report)
			sb.WriteString("*Reference Link(s)*\n\n")
			sb.WriteString(fmt.Sprintf("* https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/\n\n", version))
		}
	}

	// Reset bgcolor for future tables - exactly as in old report
	sb.WriteString("// Reset bgcolor for future tables\n[grid=none,frame=none]\n|===\n|{set:cellbgcolor!}\n|===\n\n")

	return sb.String()
}

// containsAsciiDocFormatting checks if a string already contains AsciiDoc formatting elements
func containsAsciiDocFormatting(text string) bool {
	// List of patterns that indicate AsciiDoc formatting
	patterns := []string{
		"[source,",
		"[source, ",
		"== ",
		"=== ",
		"WARNING:",
		"|===",
		"----\n",
		"[cols=",
	}

	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

// formatResultKeyForSummaryTable formats a result key for display in the summary table
func formatResultKeyForSummaryTable(resultKey types.ResultKey) string {
	cellColor := getResultKeyColor(resultKey)
	displayText := getResultKeyDisplayText(resultKey)

	return fmt.Sprintf("|{set:cellbgcolor:%s}\n%s\n", cellColor, displayText)
}

// formatResultKeyAsCenteredTable formats a result key as a centered table (as in old report)
func formatResultKeyAsCenteredTable(resultKey types.ResultKey) string {
	cellColor := getResultKeyColor(resultKey)
	displayText := getResultKeyDisplayText(resultKey)

	return fmt.Sprintf("[cols=\"^\"]\n|===\n|\n{set:cellbgcolor:%s}\n%s\n|===", cellColor, displayText)
}

// getResultKeyColor returns the color for a result key
func getResultKeyColor(resultKey types.ResultKey) string {
	switch resultKey {
	case types.ResultKeyRequired:
		return "#FF0000" // Red
	case types.ResultKeyRecommended:
		return "#FEFE20" // Yellow
	case types.ResultKeyNoChange:
		return "#00FF00" // Green
	case types.ResultKeyAdvisory:
		return "#80E5FF" // Light Blue
	case types.ResultKeyNotApplicable:
		return "#A6B9BF" // Gray
	case types.ResultKeyEvaluate:
		return "#FFFFFF" // White
	default:
		return "#FFFFFF" // White
	}
}

// getResultKeyDisplayText returns the display text for a result key
func getResultKeyDisplayText(resultKey types.ResultKey) string {
	switch resultKey {
	case types.ResultKeyRequired:
		return "Changes Required"
	case types.ResultKeyRecommended:
		return "Changes Recommended"
	case types.ResultKeyNoChange:
		return "No Change"
	case types.ResultKeyAdvisory:
		return "Advisory"
	case types.ResultKeyNotApplicable:
		return "Not Applicable"
	case types.ResultKeyEvaluate:
		return "To Be Evaluated"
	default:
		return "To Be Evaluated"
	}
}

// groupChecksByCategory organizes checks by their category
func groupChecksByCategory(checks []types.Check, results map[string]types.Result) map[types.Category][]types.Check {
	categorized := make(map[types.Category][]types.Check)

	for _, check := range checks {
		if _, exists := results[check.ID()]; exists {
			category := check.Category()

			// Map old categories to new ones for consistent reporting
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
	}

	return categorized
}
