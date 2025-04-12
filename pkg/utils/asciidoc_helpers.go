package utils

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
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
func GetStatusColor(status healthcheck.Status) string {
	switch status {
	case healthcheck.StatusOK:
		return "#00FF00" // Green
	case healthcheck.StatusWarning:
		return "#FEFE20" // Yellow
	case healthcheck.StatusCritical:
		return "#FF0000" // Red
	case healthcheck.StatusUnknown:
		return "#FFFFFF" // White
	case healthcheck.StatusNotApplicable:
		return "#A6B9BF" // Gray
	default:
		return "#FFFFFF" // White
	}
}

// GetResultKeyColor returns the color for a result key in AsciiDoc format
func GetResultKeyColor(resultKey healthcheck.ResultKey) string {
	switch resultKey {
	case healthcheck.ResultKeyNoChange:
		return "#00FF00" // Green
	case healthcheck.ResultKeyRecommended:
		return "#FEFE20" // Yellow
	case healthcheck.ResultKeyRequired:
		return "#FF0000" // Red
	case healthcheck.ResultKeyAdvisory:
		return "#80E5FF" // Light Blue
	case healthcheck.ResultKeyNotApplicable:
		return "#A6B9BF" // Gray
	case healthcheck.ResultKeyEvaluate:
		return "#FFFFFF" // White
	default:
		return "#FFFFFF" // White
	}
}

// FormatStatusCell formats a status as an AsciiDoc table cell with appropriate coloring
func FormatStatusCell(status healthcheck.Status) string {
	color := GetStatusColor(status)
	return fmt.Sprintf("|{set:cellbgcolor:%s}\n%s\n|{set:cellbgcolor!}", color, status)
}

// FormatResultKeyCell formats a result key as an AsciiDoc table cell with appropriate coloring
func FormatResultKeyCell(resultKey healthcheck.ResultKey) string {
	color := GetResultKeyColor(resultKey)
	switch resultKey {
	case healthcheck.ResultKeyNoChange:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nNo Change\n|{set:cellbgcolor!}", color)
	case healthcheck.ResultKeyRecommended:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nChanges Recommended\n|{set:cellbgcolor!}", color)
	case healthcheck.ResultKeyRequired:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nChanges Required\n|{set:cellbgcolor!}", color)
	case healthcheck.ResultKeyAdvisory:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nAdvisory\n|{set:cellbgcolor!}", color)
	case healthcheck.ResultKeyNotApplicable:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nNot Applicable\n|{set:cellbgcolor!}", color)
	case healthcheck.ResultKeyEvaluate:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nTo Be Evaluated\n|{set:cellbgcolor!}", color)
	default:
		return fmt.Sprintf("|{set:cellbgcolor:%s}\nUnknown\n|{set:cellbgcolor!}", color)
	}
}

// GetChanges returns a formatted AsciiDoc table for a result key
// This replicates the original GetChanges function from the original codebase
func GetChanges(resultKey healthcheck.ResultKey) string {
	options := map[healthcheck.ResultKey]string{
		healthcheck.ResultKeyRequired: `[cols="^"] 
|===
|
{set:cellbgcolor:#FF0000}
Changes Required
|===`,
		healthcheck.ResultKeyRecommended: `[cols="^"] 
|===
|
{set:cellbgcolor:#FEFE20}
Changes Recommended
|===`,
		healthcheck.ResultKeyNoChange: `[cols="^"] 
|===
|
{set:cellbgcolor:#00FF00}
No Change
|===`,
		healthcheck.ResultKeyAdvisory: `[cols="^"] 
|===
|
{set:cellbgcolor:#80E5FF}
Advisory
|===`,
		healthcheck.ResultKeyEvaluate: `[cols="^"] 
|===
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
|===`,
		healthcheck.ResultKeyNotApplicable: `[cols="^"] 
|===
|
{set:cellbgcolor:#A6B9BF}
Not Applicable
|===`,
	}

	result, ok := options[resultKey]
	if !ok {
		return options[healthcheck.ResultKeyEvaluate]
	}
	return result
}

// GetKeyChanges returns a formatted AsciiDoc table cell for a result key
// This replicates the original GetKeyChanges function from the original codebase
func GetKeyChanges(resultKey healthcheck.ResultKey) string {
	options := map[healthcheck.ResultKey]string{
		healthcheck.ResultKeyRequired: `| 
{set:cellbgcolor:#FF0000}
Changes Required
`,
		healthcheck.ResultKeyRecommended: `| 
{set:cellbgcolor:#FEFE20}
Changes Recommended
`,
		healthcheck.ResultKeyNoChange: `| 
{set:cellbgcolor:#00FF00}
No Change
`,
		healthcheck.ResultKeyAdvisory: `| 
{set:cellbgcolor:#80E5FF}
Advisory
`,
		healthcheck.ResultKeyNotApplicable: `| 
{set:cellbgcolor:#A6B9BF}
Not Applicable
`,
		healthcheck.ResultKeyEvaluate: `| 
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
`,
	}

	result, ok := options[resultKey]
	if !ok {
		return options[healthcheck.ResultKeyEvaluate]
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
func GenerateAsciiDocCheckSection(check healthcheck.Check, result healthcheck.Result, version string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("== %s\n\n", check.Name()))
	sb.WriteString(GetChanges(result.ResultKey) + "\n\n")

	if result.Detail != "" {
		sb.WriteString("[source,bash]\n----\n")
		sb.WriteString(result.Detail)
		sb.WriteString("\n----\n\n")
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

	sb.WriteString("**Reference Link(s)**\n\n")
	sb.WriteString(fmt.Sprintf("* https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/\n", version))

	return sb.String()
}

// GenerateAsciiDocSummaryTable generates the summary table for all checks
func GenerateAsciiDocSummaryTable(checks []healthcheck.Check, results map[string]healthcheck.Result) string {
	var sb strings.Builder

	sb.WriteString("= Summary\n\n")
	sb.WriteString("[cols=\"1,2,2,3\", options=header]\n|===\n|*Category*\n|*Item Evaluated*\n|*Observed Result*\n|*Recommendation*\n\n")

	for _, check := range checks {
		result, exists := results[check.ID()]
		if !exists {
			continue
		}

		// Category
		sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

		// Item Evaluated
		sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

		// Observed Result
		sb.WriteString("|\n" + result.Message + "\n\n")

		// Recommendation
		sb.WriteString(GetKeyChanges(result.ResultKey) + "\n\n")
	}

	sb.WriteString("|===\n\n")
	return sb.String()
}

// GenerateAsciiDocCategorySections generates sections for each category
func GenerateAsciiDocCategorySections(categorizedChecks map[healthcheck.Category][]healthcheck.Check, results map[string]healthcheck.Result) string {
	var sb strings.Builder

	// Sort categories
	categories := []healthcheck.Category{
		healthcheck.CategoryCluster,
		healthcheck.CategorySecurity,
		healthcheck.CategoryNetworking,
		healthcheck.CategoryStorage,
		healthcheck.CategoryApplications,
		healthcheck.CategoryMonitoring,
		healthcheck.CategoryInfrastructure,
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
			sb.WriteString("|\n{set:cellbgcolor!}\n" + string(check.Category()) + "\n\n")

			// Item Evaluated
			sb.WriteString("a|\n<<" + check.Name() + ">>\n\n")

			// Observed Result
			sb.WriteString("|\n" + result.Message + "\n\n")

			// Recommendation
			sb.WriteString(GetKeyChanges(result.ResultKey) + "\n\n")
		}

		sb.WriteString("|===\n\n")
	}

	return sb.String()
}

// GenerateFullAsciiDocReport generates a complete AsciiDoc report for all health checks
func GenerateFullAsciiDocReport(title string, checks []healthcheck.Check, results map[string]healthcheck.Result) string {
	var sb strings.Builder

	// Get OpenShift version for documentation links
	version, err := GetOpenShiftMajorMinorVersion()
	if err != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Generate report header
	sb.WriteString(GenerateAsciiDocReportHeader(title))

	// Generate summary table
	sb.WriteString(GenerateAsciiDocSummaryTable(checks, results))

	// Organize checks by category
	categorizedChecks := make(map[healthcheck.Category][]healthcheck.Check)
	for _, check := range checks {
		category := check.Category()
		categorizedChecks[category] = append(categorizedChecks[category], check)
	}

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
