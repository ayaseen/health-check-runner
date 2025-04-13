package healthcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// EnhancedReporter provides enhanced AsciiDoc reporting capabilities
// that match the format of the original health-check runner
type EnhancedReporter struct {
	config ReportConfig
	runner *Runner
}

// NewEnhancedReporter creates a new enhanced reporter
func NewEnhancedReporter(config ReportConfig, runner *Runner) *EnhancedReporter {
	return &EnhancedReporter{
		config: config,
		runner: runner,
	}
}

// Generate generates a report in the enhanced format
func (r *EnhancedReporter) Generate() (string, error) {
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
	case FormatAsciiDoc:
		content, err = r.generateEnhancedAsciiDoc()
	case FormatHTML:
		// Fall back to standard HTML for now
		reporter := NewReporter(r.config, r.runner)
		return reporter.Generate()
	case FormatJSON:
		// Fall back to standard JSON for now
		reporter := NewReporter(r.config, r.runner)
		return reporter.Generate()
	case FormatSummary:
		// Fall back to standard summary for now
		reporter := NewReporter(r.config, r.runner)
		return reporter.Generate()
	default:
		return "", fmt.Errorf("unsupported report format: %s", r.config.Format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// Write content to file
	filepath := filepath.Join(r.config.OutputDir, filename)
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return filepath, nil
}

// getFilename returns the filename for the report
func (r *EnhancedReporter) getFilename() string {
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
		case FormatAsciiDoc:
			filename += ".adoc"
		case FormatHTML:
			filename += ".html"
		case FormatJSON:
			filename += ".json"
		case FormatSummary:
			filename += ".txt"
		}
	}

	return filename
}

// generateEnhancedAsciiDoc generates an enhanced AsciiDoc report
func (r *EnhancedReporter) generateEnhancedAsciiDoc() (string, error) {
	checks := r.runner.GetChecks()
	results := r.runner.GetResults()

	// Generate the full report using the enhanced template format
	content := utils.GenerateFullAsciiDocReport(r.config.Title, checks, results)

	return content, nil
}
