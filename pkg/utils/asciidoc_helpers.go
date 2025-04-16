/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file provides utilities for AsciiDoc report formatting. It includes:

- Functions for creating color-coded status indicators
- Methods for generating formatted tables and sections
- Utilities for consistent styling and presentation
- Helpers for creating report headers and structured content
- Functions for formatting result keys and recommendations

These utilities help create readable, visually appealing reports that clearly communicate health check findings.
*/

package utils

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// AsciiDocFormatOptions defines options for AsciiDoc formatting
type AsciiDocFormatOptions struct {
	IncludeTimestamp       bool
	IncludeDetailedResults bool
	Title                  string
	GroupByCategory        bool
	ColorOutput            bool
}

// GetStatusColor returns the color for a status in AsciiDoc format
func GetStatusColor(status types.Status) string {
	switch status {
	case types.StatusOK:
		return "#00FF00" // Green
	case types.StatusWarning:
		return "#FEFE20" // Yellow
	case types.StatusCritical:
		return "#FF0000" // Red
	case types.StatusUnknown:
		return "#FFFFFF" // White
	case types.StatusNotApplicable:
		return "#A6B9BF" // Gray
	default:
		return "#FFFFFF" // White
	}
}

// GetResultKeyColor returns the color for a result key in AsciiDoc format
func GetResultKeyColor(resultKey types.ResultKey) string {
	switch resultKey {
	case types.ResultKeyNoChange:
		return "#00FF00" // Green
	case types.ResultKeyRecommended:
		return "#FEFE20" // Yellow
	case types.ResultKeyRequired:
		return "#FF0000" // Red
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

// FormatStatusCell formats a status as an AsciiDoc table cell with appropriate coloring
func FormatStatusCell(status types.Status) string {
	color := GetStatusColor(status)
	return fmt.Sprintf("|{set:cellbgcolor:%s}\n%s\n|{set:cellbgcolor!}", color, status)
}

// FormatResultKeyCell formats a result key as an AsciiDoc table cell with appropriate coloring
func FormatResultKeyCell(resultKey types.ResultKey) string {
	color := GetResultKeyColor(resultKey)
	switch resultKey {
	case types.ResultKeyNoChange:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nNo Change\n|{set:cellbgcolor!}", color)
	case types.ResultKeyRecommended:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nChanges Recommended\n|{set:cellbgcolor!}", color)
	case types.ResultKeyRequired:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nChanges Required\n|{set:cellbgcolor!}", color)
	case types.ResultKeyAdvisory:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nAdvisory\n|{set:cellbgcolor!}", color)
	case types.ResultKeyNotApplicable:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nNot Applicable\n|{set:cellbgcolor!}", color)
	case types.ResultKeyEvaluate:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nTo Be Evaluated\n|{set:cellbgcolor!}", color)
	default:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nUnknown\n|{set:cellbgcolor!}", color)
	}
}

// GetChanges returns a formatted AsciiDoc table for a result key
func GetChanges(resultKey types.ResultKey) string {
	options := map[types.ResultKey]string{
		types.ResultKeyRequired: `[cols="^"] 
|===
|
{set:cellbgcolor:#FF0000}
Changes Required
|===`,
		types.ResultKeyRecommended: `[cols="^"] 
|===
|
{set:cellbgcolor:#FEFE20}
Changes Recommended
|===`,
		types.ResultKeyNoChange: `[cols="^"] 
|===
|
{set:cellbgcolor:#00FF00}
No Change
|===`,
		types.ResultKeyAdvisory: `[cols="^"] 
|===
|
{set:cellbgcolor:#80E5FF}
Advisory
|===`,
		types.ResultKeyEvaluate: `[cols="^"] 
|===
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
|===`,
		types.ResultKeyNotApplicable: `[cols="^"] 
|===
|
{set:cellbgcolor:#A6B9BF}
Not Applicable
|===`,
	}

	result, ok := options[resultKey]
	if !ok {
		return options[types.ResultKeyEvaluate]
	}
	return result
}

// GetKeyChanges returns a formatted AsciiDoc table cell for a result key
func GetKeyChanges(resultKey types.ResultKey) string {
	options := map[types.ResultKey]string{
		types.ResultKeyRequired: `| 
{set:cellbgcolor:#FF0000}
Changes Required
`,
		types.ResultKeyRecommended: `| 
{set:cellbgcolor:#FEFE20}
Changes Recommended
`,
		types.ResultKeyNoChange: `| 
{set:cellbgcolor:#00FF00}
No Change
`,
		types.ResultKeyAdvisory: `| 
{set:cellbgcolor:#80E5FF}
Advisory
`,
		types.ResultKeyNotApplicable: `| 
{set:cellbgcolor:#A6B9BF}
Not Applicable
`,
		types.ResultKeyEvaluate: `| 
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
`,
	}

	result, ok := options[resultKey]
	if !ok {
		return options[types.ResultKeyEvaluate]
	}
	return result
}

// GenerateAsciiDocReportHeader generates the AsciiDoc header for the report
func GenerateAsciiDocReportHeader(title string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("= %s\n\n", title))
	sb.WriteString("ifdef::env-github[]\n:tip-caption: :bulb:\n:note-caption: :information_source:\n:important-caption: :heavy_exclamation_mark:\n:caution-caption: :fire:\n:warning-caption: :warning:\nendif::[]\n\n")

	// Add key for status colors
	sb.WriteString("= Key\n\n")
	sb.WriteString("[cols=\"1,3\", options=header]\n|===\n|Value\n|Description\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FF0000}\nChanges Required\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Indicates Changes Required for system stability, subscription compliance, or other reason.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FEFE20}\nChanges Recommended\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Indicates Changes Recommended to align with recommended practices, but not urgently required\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#A6B9BF}\nN/A\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No advise given on line item. For line items which are data-only to provide context.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#80E5FF}\nAdvisory\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No change required or recommended, but additional information provided.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#00FF00}\nNo Change\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("No change required. In alignment with recommended practices.\n\n")

	sb.WriteString("|\n{set:cellbgcolor:#FFFFFF}\nTo Be Evaluated\n|\n{set:cellbgcolor!}\n")
	sb.WriteString("Not yet evaluated. Will appear only in draft copies.\n|===\n\n")

	return sb.String()
}

// GenerateAsciiDocCheckSection generates a detailed section for a health check
func GenerateAsciiDocCheckSection(check types.Check, result types.Result, version string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("== %s\n\n", check.Name()))
	sb.WriteString(GetChanges(result.ResultKey) + "\n\n")

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
			// Otherwise, wrap it in a source block
			sb.WriteString("[source, bash]\n----\n")
			sb.WriteString(result.Detail)
			sb.WriteString("\n----\n\n")
		}
	}

	sb.WriteString("**Observation**\n\n")
	sb.WriteString(result.Message + "\n\n")

	sb.WriteString("**Recommendation**\n\n")
	if len(result.Recommendations) > 0 {
		for _, rec := range result.Recommendations {
			sb.WriteString(rec + "\n\n")
		}
	} else {
		sb.WriteString("None.\n\n")
	}

	sb.WriteString("*Reference Link(s)*\n\n")
	sb.WriteString(fmt.Sprintf("* https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/\n", version))

	return sb.String()
}

// GenerateAsciiDocSummaryTable generates the summary table for all checks
func GenerateAsciiDocSummaryTable(checks []types.Check, results map[string]types.Result) string {
	var sb strings.Builder

	sb.WriteString("= Summary\n\n")
	sb.WriteString("[cols=\"1,2,2,3\", options=header]\n|===\n|*Category*\n|*Item Evaluated*\n|*Observed Result*\n|*Recommendation*\n\n")

	for _, check := range checks {
		result, exists := results[check.ID()]
		if !exists {
			continue
		}

		// Category
		sb.WriteString("// ------------------------ITEM START\n")
		sb.WriteString("// ----ITEM SOURCE:  ./content/healthcheck-items/" + check.ID() + ".item\n\n")
		sb.WriteString("// Category\n")
		sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

		// Item Evaluated
		sb.WriteString("// Item Evaluated\n")
		sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

		// Observed Result
		sb.WriteString("| " + result.Message + " \n\n")

		// Recommendation
		sb.WriteString(GetKeyChanges(result.ResultKey) + "\n\n")
		sb.WriteString("// ------------------------ITEM END\n\n")
	}

	sb.WriteString("|===\n\n")
	return sb.String()
}

// GenerateAsciiDocCategorySections generates sections for each category
func GenerateAsciiDocCategorySections(
	categorizedChecks map[types.Category][]types.Check,
	results map[string]types.Result,
) string {
	var sb strings.Builder

	// Sort categories in the same order as the old report
	categories := []types.Category{
		types.CategoryInfra,
		types.CategoryNetwork,
		types.CategoryStorage,
		types.CategoryClusterConfig,
		types.CategoryAppDev,
		types.CategorySecurity,
		types.CategoryOpReady,
	}

	for _, category := range categories {
		checks, exists := categorizedChecks[category]
		if !exists || len(checks) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("<<<\n\n{set:cellbgcolor!}\n\n# %s\n\n", category))
		sb.WriteString("[cols=\"1,2,2,3\", options=header]\n|===\n|*Category*\n|*Item Evaluated*\n|*Observed Result*\n|*Recommendation*\n\n")

		for _, check := range checks {
			result, exists := results[check.ID()]
			if !exists {
				continue
			}

			// Category
			sb.WriteString("// ------------------------ITEM START\n")
			sb.WriteString("// ----ITEM SOURCE:  ./content/healthcheck-items/" + check.ID() + ".item\n\n")
			sb.WriteString("// Category\n")
			sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

			// Item Evaluated
			sb.WriteString("// Item Evaluated\n")
			sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

			// Observed Result
			sb.WriteString("| " + result.Message + " \n\n")

			// Recommendation
			sb.WriteString(GetKeyChanges(result.ResultKey) + "\n\n")
			sb.WriteString("// ------------------------ITEM END\n\n")
		}

		sb.WriteString("|===\n\n")
	}

	return sb.String()
}
