/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file defines core types and constants used throughout the health check framework. It includes:

- Status types for health check results (OK, Warning, Critical, etc.)
- Result key types for indicating the importance of findings
- Category types for organizing health checks
- Report format types for different output formats
- Core interfaces and structures for representing check results
- Configuration types for report generation

These types provide a standardized way to represent and process health check data throughout the application.
*/

package types

// Status represents the result status of a health check
type Status string

const (
	// StatusOK indicates everything is working correctly
	StatusOK Status = "OK"

	// StatusWarning indicates a potential issue that should be addressed
	StatusWarning Status = "Warning"

	// StatusCritical indicates a critical issue that requires immediate attention
	StatusCritical Status = "Critical"

	// StatusUnknown indicates the status could not be determined
	StatusUnknown Status = "Unknown"

	// StatusNotApplicable indicates the check does not apply to this environment
	StatusNotApplicable Status = "NotApplicable"
)

// ResultKey represents the level of importance for a result in a report summary
type ResultKey string

const (
	// ResultKeyNoChange indicates no changes are needed
	ResultKeyNoChange ResultKey = "nochange"

	// ResultKeyRecommended indicates changes are recommended
	ResultKeyRecommended ResultKey = "recommended"

	// ResultKeyRequired indicates changes are required
	ResultKeyRequired ResultKey = "required"

	// ResultKeyAdvisory indicates additional information
	ResultKeyAdvisory ResultKey = "advisory"

	// ResultKeyNotApplicable indicates the check does not apply
	ResultKeyNotApplicable ResultKey = "na"

	// ResultKeyEvaluate indicates the result needs evaluation
	ResultKeyEvaluate ResultKey = "eval"
)

// Category represents a category of health checks
type Category string

const (
	// CategoryClusterConfig is for cluster-level checks
	CategoryClusterConfig Category = "Cluster Config"

	// CategoryNetworking is for networking-related checks
	CategoryNetworking Category = "Networking"

	// CategoryApplications is for application-related checks
	CategoryApplications Category = "Applications"

	// CategoryOpReady is for operational readiness checks
	CategoryOpReady Category = "Op-Ready"

	// CategorySecurity is for security-related checks
	CategorySecurity Category = "Security"

	// CategoryStorage is for storage-related checks
	CategoryStorage Category = "Storage"

	// CategoryPerformance is for performance-related checks
	CategoryPerformance Category = "Performance"
)

// ReportFormat defines the format of the generated report
type ReportFormat string

const (
	// FormatAsciiDoc generates an AsciiDoc report
	FormatAsciiDoc ReportFormat = "asciidoc"

	// FormatHTML generates an HTML report
	FormatHTML ReportFormat = "html"

	// FormatJSON generates a JSON report
	FormatJSON ReportFormat = "json"

	// FormatSummary generates a brief summary
	FormatSummary ReportFormat = "summary"
)

// Result represents the result of a health check
type Result struct {
	// CheckID is the unique identifier of the health check
	CheckID string

	// Status indicates the result status (OK, Warning, Critical, etc.)
	Status Status

	// Message is a brief description of the result
	Message string

	// ResultKey indicates the importance of the result in a report
	ResultKey ResultKey

	// Detail provides detailed information about the result
	Detail string

	// Recommendations are suggestions to address any issues
	Recommendations []string

	// ExecutionTime is how long the check took to run
	ExecutionTime string

	// Metadata is additional contextual information
	Metadata map[string]string
}

// Check defines the interface for a health check
type Check interface {
	// ID returns a unique identifier for the check
	ID() string

	// Name returns a human-readable name for the check
	Name() string

	// Description returns a description of what the check does
	Description() string

	// Category returns the category the check belongs to
	Category() Category
}

// ReportConfig defines the configuration for report generation
type ReportConfig struct {
	// Format is the report format to generate
	Format ReportFormat

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
