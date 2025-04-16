/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements the reporting functionality for health checks. It:

- Generates formatted reports of health check results
- Supports multiple output formats (AsciiDoc, HTML, JSON, text summary)
- Organizes results by category and status
- Includes detailed result information and recommendations
- Handles report file creation and formatting options

This component transforms raw health check results into readable, actionable reports for administrators.
*/

package healthcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// ReportConfig defines the configuration for report generation
type ReportConfig struct {
	// Format is the report format to generate
	Format types.ReportFormat

	// OutputDir is where the report will be saved
	OutputDir string

	// Filename is the name of the report file
	Filename string

	// IncludeTimestamp adds a timestamp to the filename
	IncludeTimestamp bool

	// IncludeDetailedResults includes detailed results in the report
	IncludeDetailedResults bool

	// Title is the title of the report
	Title string

	// GroupByCategory groups results by category
	GroupByCategory bool

	// ColorOutput enables colored output for terminal formats
	ColorOutput bool

	// UseEnhancedAsciiDoc enables enhanced AsciiDoc formatting
	UseEnhancedAsciiDoc bool
}

// ReportGeneratorFunc is a function type for generating reports
type ReportGeneratorFunc func(title string, checks []types.Check, results map[string]types.Result) string

// Reporter generates reports for health check results
type Reporter struct {
	config ReportConfig
	runner *Runner
	// Generator function for enhanced AsciiDoc reports
	enhancedAsciiDocGenerator ReportGeneratorFunc
}

// NewReporter creates a new reporter
func NewReporter(config ReportConfig, runner *Runner) *Reporter {
	return &Reporter{
		config: config,
		runner: runner,
		// This will be set in SetEnhancedAsciiDocGenerator
		enhancedAsciiDocGenerator: nil,
	}
}

// SetEnhancedAsciiDocGenerator sets the function to generate enhanced AsciiDoc reports
func (r *Reporter) SetEnhancedAsciiDocGenerator(generator ReportGeneratorFunc) {
	r.enhancedAsciiDocGenerator = generator
}

// Generate generates a report
func (r *Reporter) Generate() (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := r.getFilename()

	// Generate content based on format
	var content string
	var err error

	switch r.config.Format {
	case types.FormatAsciiDoc:
		if r.config.UseEnhancedAsciiDoc && r.enhancedAsciiDocGenerator != nil {
			// Convert runner results to types.Result
			typesResults := make(map[string]types.Result)
			for id, result := range r.runner.results {
				typesResults[id] = result.ToTypesResult()
			}

			// Convert checks to types.Check
			var typesChecks []types.Check
			for _, check := range r.runner.checks {
				typesChecks = append(typesChecks, check)
			}

			content = r.enhancedAsciiDocGenerator(r.config.Title, typesChecks, typesResults)
		} else {
			content, err = r.generateAsciiDoc()
		}
	case types.FormatHTML:
		content, err = r.generateHTML()
	case types.FormatJSON:
		content, err = r.generateJSON()
	case types.FormatSummary:
		content, err = r.generateSummary()
	default:
		return "", fmt.Errorf("unsupported report format: %s", r.config.Format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// Write content to file
	outputPath := filepath.Join(r.config.OutputDir, filename)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return outputPath, nil
}

// getFilename returns the filename for the report
func (r *Reporter) getFilename() string {
	filename := r.config.Filename

	// Add timestamp if requested
	if r.config.IncludeTimestamp {
		timestamp := time.Now().Format("20060102-150405")

		// Insert timestamp before extension
		if ext := filepath.Ext(filename); ext != "" {
			filename = filename[:len(filename)-len(ext)] + "-" + timestamp + ext
		} else {
			filename = filename + "-" + timestamp
		}
	}

	// Add extension based on format if not present
	ext := filepath.Ext(filename)
	if ext == "" {
		switch r.config.Format {
		case types.FormatAsciiDoc:
			filename += ".adoc"
		case types.FormatHTML:
			filename += ".html"
		case types.FormatJSON:
			filename += ".json"
		case types.FormatSummary:
			filename += ".txt"
		}
	}

	return filename
}

// generateAsciiDoc generates an AsciiDoc report (standard format)
func (r *Reporter) generateAsciiDoc() (string, error) {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("= %s\n\n", r.config.Title))
	sb.WriteString("ifdef::env-github[]\n:tip-caption: :bulb:\n:note-caption: :information_source:\n:important-caption: :heavy_exclamation_mark:\n:caution-caption: :fire:\n:warning-caption: :warning:\nendif::[]\n\n")

	// Summary
	sb.WriteString("== Summary\n\n")

	// Add counts by status
	counts := r.runner.CountByStatus()
	sb.WriteString("[cols=\"1,1\", options=\"header\"]\n|===\n|Status|Count\n\n")

	for _, status := range []types.Status{types.StatusOK, types.StatusWarning, types.StatusCritical, types.StatusUnknown, types.StatusNotApplicable} {
		count := counts[status]
		sb.WriteString(fmt.Sprintf("|%s|%d\n", status, count))
	}
	sb.WriteString("|===\n\n")

	// Generate table of all checks
	sb.WriteString("== Health Check Results\n\n")

	if r.config.GroupByCategory {
		resultsByCategory := r.runner.GetResultsByCategory()

		for category, results := range resultsByCategory {
			sb.WriteString(fmt.Sprintf("=== %s\n\n", category))
			r.writeAsciiDocResultsTable(&sb, results)
			sb.WriteString("\n")
		}
	} else {
		var allResults []Result
		for _, result := range r.runner.GetResults() {
			allResults = append(allResults, result)
		}
		r.writeAsciiDocResultsTable(&sb, allResults)
	}

	// Add detailed results if requested
	if r.config.IncludeDetailedResults {
		sb.WriteString("== Detailed Results\n\n")

		for _, check := range r.runner.checks {
			if result, exists := r.runner.results[check.ID()]; exists {
				sb.WriteString(fmt.Sprintf("=== %s\n\n", check.Name()))
				sb.WriteString(fmt.Sprintf("*ID:* %s\n\n", check.ID()))
				sb.WriteString(fmt.Sprintf("*Description:* %s\n\n", check.Description()))
				sb.WriteString(fmt.Sprintf("*Category:* %s\n\n", check.Category()))
				sb.WriteString(fmt.Sprintf("*Status:* %s\n\n", result.Status))
				sb.WriteString(fmt.Sprintf("*Message:* %s\n\n", result.Message))

				if len(result.Recommendations) > 0 {
					sb.WriteString("*Recommendations:*\n\n")
					for _, rec := range result.Recommendations {
						sb.WriteString(fmt.Sprintf("* %s\n", rec))
					}
					sb.WriteString("\n")
				}

				if result.Detail != "" {
					sb.WriteString("*Details:*\n\n")
					sb.WriteString("----\n")
					sb.WriteString(result.Detail)
					sb.WriteString("\n----\n\n")
				}

				sb.WriteString(fmt.Sprintf("*Execution Time:* %s\n\n", result.ExecutionTime))
				sb.WriteString("'''\n\n")
			}
		}
	}

	return sb.String(), nil
}

// writeAsciiDocResultsTable writes a table of results in AsciiDoc format
func (r *Reporter) writeAsciiDocResultsTable(sb *strings.Builder, results []Result) {
	sb.WriteString("[cols=\"1,3,1,3\", options=\"header\"]\n|===\n|Check|Result|Status|Recommendations\n\n")

	for _, result := range results {
		// Find the check this result belongs to
		var checkName string
		for _, check := range r.runner.checks {
			if check.ID() == result.CheckID {
				checkName = check.Name()
				break
			}
		}

		// Format recommendations
		var recommendations string
		if len(result.Recommendations) > 0 {
			for _, rec := range result.Recommendations {
				recommendations += fmt.Sprintf("* %s\n", rec)
			}
		} else {
			recommendations = "None"
		}

		sb.WriteString(fmt.Sprintf("|%s|%s|%s|%s\n",
			checkName,
			result.Message,
			result.Status,
			recommendations))
	}

	sb.WriteString("|===\n")
}

// generateHTML generates an HTML report
func (r *Reporter) generateHTML() (string, error) {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n")
	sb.WriteString("<html>\n<head>\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", r.config.Title))
	sb.WriteString("<style>\n")
	sb.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
	sb.WriteString("h1 { color: #333; }\n")
	sb.WriteString("table { border-collapse: collapse; width: 100%; margin-bottom: 20px; }\n")
	sb.WriteString("th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }\n")
	sb.WriteString("th { background-color: #f2f2f2; }\n")
	sb.WriteString(".status-OK { color: green; }\n")
	sb.WriteString(".status-Warning { color: orange; }\n")
	sb.WriteString(".status-Critical { color: red; }\n")
	sb.WriteString(".status-Unknown { color: gray; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")

	// Title
	sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", r.config.Title))

	// Summary
	sb.WriteString("<h2>Summary</h2>\n")

	// Add counts by status
	counts := r.runner.CountByStatus()
	sb.WriteString("<table>\n<tr><th>Status</th><th>Count</th></tr>\n")

	for _, status := range []types.Status{types.StatusOK, types.StatusWarning, types.StatusCritical, types.StatusUnknown, types.StatusNotApplicable} {
		count := counts[status]
		sb.WriteString(fmt.Sprintf("<tr><td class=\"status-%s\">%s</td><td>%d</td></tr>\n", status, status, count))
	}
	sb.WriteString("</table>\n")

	// Generate table of all checks
	sb.WriteString("<h2>Health Check Results</h2>\n")

	if r.config.GroupByCategory {
		resultsByCategory := r.runner.GetResultsByCategory()

		for category, results := range resultsByCategory {
			sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", category))
			r.writeHTMLResultsTable(&sb, results)
		}
	} else {
		var allResults []Result
		for _, result := range r.runner.GetResults() {
			allResults = append(allResults, result)
		}
		r.writeHTMLResultsTable(&sb, allResults)
	}

	// Add detailed results if requested
	if r.config.IncludeDetailedResults {
		sb.WriteString("<h2>Detailed Results</h2>\n")

		for _, check := range r.runner.checks {
			if result, exists := r.runner.results[check.ID()]; exists {
				sb.WriteString(fmt.Sprintf("<h3>%s</h3>\n", check.Name()))
				sb.WriteString("<dl>\n")
				sb.WriteString(fmt.Sprintf("<dt>ID</dt><dd>%s</dd>\n", check.ID()))
				sb.WriteString(fmt.Sprintf("<dt>Description</dt><dd>%s</dd>\n", check.Description()))
				sb.WriteString(fmt.Sprintf("<dt>Category</dt><dd>%s</dd>\n", check.Category()))
				sb.WriteString(fmt.Sprintf("<dt>Status</dt><dd class=\"status-%s\">%s</dd>\n", result.Status, result.Status))
				sb.WriteString(fmt.Sprintf("<dt>Message</dt><dd>%s</dd>\n", result.Message))

				if len(result.Recommendations) > 0 {
					sb.WriteString("<dt>Recommendations</dt><dd><ul>\n")
					for _, rec := range result.Recommendations {
						sb.WriteString(fmt.Sprintf("<li>%s</li>\n", rec))
					}
					sb.WriteString("</ul></dd>\n")
				}

				if result.Detail != "" {
					sb.WriteString("<dt>Details</dt><dd><pre>")
					sb.WriteString(result.Detail)
					sb.WriteString("</pre></dd>\n")
				}

				sb.WriteString(fmt.Sprintf("<dt>Execution Time</dt><dd>%s</dd>\n", result.ExecutionTime))
				sb.WriteString("</dl>\n")
				sb.WriteString("<hr>\n")
			}
		}
	}

	sb.WriteString("</body>\n</html>")

	return sb.String(), nil
}

// writeHTMLResultsTable writes a table of results in HTML format
func (r *Reporter) writeHTMLResultsTable(sb *strings.Builder, results []Result) {
	sb.WriteString("<table>\n<tr><th>Check</th><th>Result</th><th>Status</th><th>Recommendations</th></tr>\n")

	for _, result := range results {
		// Find the check this result belongs to
		var checkName string
		for _, check := range r.runner.checks {
			if check.ID() == result.CheckID {
				checkName = check.Name()
				break
			}
		}

		// Format recommendations
		var recommendations string
		if len(result.Recommendations) > 0 {
			recommendations = "<ul>"
			for _, rec := range result.Recommendations {
				recommendations += fmt.Sprintf("<li>%s</li>", rec)
			}
			recommendations += "</ul>"
		} else {
			recommendations = "None"
		}

		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td class=\"status-%s\">%s</td><td>%s</td></tr>\n",
			checkName,
			result.Message,
			result.Status,
			result.Status,
			recommendations))
	}

	sb.WriteString("</table>\n")
}

// generateJSON generates a JSON report
func (r *Reporter) generateJSON() (string, error) {
	type jsonResult struct {
		CheckID         string            `json:"check_id"`
		CheckName       string            `json:"check_name"`
		Description     string            `json:"description"`
		Category        string            `json:"category"`
		Status          string            `json:"status"`
		Message         string            `json:"message"`
		ResultKey       string            `json:"result_key"`
		Detail          string            `json:"detail,omitempty"`
		Recommendations []string          `json:"recommendations,omitempty"`
		ExecutionTime   string            `json:"execution_time"`
		Metadata        map[string]string `json:"metadata,omitempty"`
	}

	type jsonReport struct {
		Title           string         `json:"title"`
		GeneratedAt     string         `json:"generated_at"`
		ResultsByStatus map[string]int `json:"results_by_status"`
		Results         []jsonResult   `json:"results"`
	}

	// Convert results to JSON format
	var jsonResults []jsonResult

	for _, check := range r.runner.checks {
		if result, exists := r.runner.results[check.ID()]; exists {
			jsonResults = append(jsonResults, jsonResult{
				CheckID:         check.ID(),
				CheckName:       check.Name(),
				Description:     check.Description(),
				Category:        string(check.Category()),
				Status:          string(result.Status),
				Message:         result.Message,
				ResultKey:       string(result.ResultKey),
				Detail:          result.Detail,
				Recommendations: result.Recommendations,
				ExecutionTime:   result.ExecutionTime.String(),
				Metadata:        result.Metadata,
			})
		}
	}

	// Create report
	counts := r.runner.CountByStatus()
	countsByStatus := make(map[string]int)

	for status, count := range counts {
		countsByStatus[string(status)] = count
	}

	report := jsonReport{
		Title:           r.config.Title,
		GeneratedAt:     time.Now().Format(time.RFC3339),
		ResultsByStatus: countsByStatus,
		Results:         jsonResults,
	}

	// Marshal report to JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	return string(jsonData), nil
}

// generateSummary generates a summary report
func (r *Reporter) generateSummary() (string, error) {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("%s\n", r.config.Title))
	sb.WriteString(strings.Repeat("=", len(r.config.Title)))
	sb.WriteString("\n\n")

	// Summary
	sb.WriteString("Summary:\n")

	// Add counts by status
	counts := r.runner.CountByStatus()

	for _, status := range []types.Status{types.StatusOK, types.StatusWarning, types.StatusCritical, types.StatusUnknown, types.StatusNotApplicable} {
		count := counts[status]
		statusStr := string(status)

		if r.config.ColorOutput {
			switch status {
			case types.StatusOK:
				statusStr = "\033[32m" + statusStr + "\033[0m" // Green
			case types.StatusWarning:
				statusStr = "\033[33m" + statusStr + "\033[0m" // Yellow
			case types.StatusCritical:
				statusStr = "\033[31m" + statusStr + "\033[0m" // Red
			case types.StatusUnknown:
				statusStr = "\033[37m" + statusStr + "\033[0m" // Light gray
			}
		}

		sb.WriteString(fmt.Sprintf("- %s: %d\n", statusStr, count))
	}

	sb.WriteString("\n")

	// List checks with issues
	var issueCount int

	for _, check := range r.runner.checks {
		if result, exists := r.runner.results[check.ID()]; exists {
			if result.Status == types.StatusWarning || result.Status == types.StatusCritical {
				issueCount++

				statusStr := string(result.Status)
				if r.config.ColorOutput {
					switch result.Status {
					case types.StatusWarning:
						statusStr = "\033[33m" + statusStr + "\033[0m" // Yellow
					case types.StatusCritical:
						statusStr = "\033[31m" + statusStr + "\033[0m" // Red
					}
				}

				sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", statusStr, check.Name(), result.Message))

				if len(result.Recommendations) > 0 {
					sb.WriteString("  Recommendations:\n")
					for _, rec := range result.Recommendations {
						sb.WriteString(fmt.Sprintf("  - %s\n", rec))
					}
				}

				sb.WriteString("\n")
			}
		}
	}

	if issueCount == 0 {
		sb.WriteString("No issues found.\n")
	}

	return sb.String(), nil
}
