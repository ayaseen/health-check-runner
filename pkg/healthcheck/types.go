// pkg/healthcheck/types.go

package healthcheck

import (
	"fmt"
	"time"
)

// Status represents the health check status
type Status string

const (
	// StatusOK indicates the check passed without issues
	StatusOK Status = "OK"

	// StatusWarning indicates the check passed but with warnings
	StatusWarning Status = "Warning"

	// StatusCritical indicates the check failed with critical issues
	StatusCritical Status = "Critical"

	// StatusInfo indicates the check is informational only
	StatusInfo Status = "Info"

	// StatusNotApplicable indicates the check is not applicable to the environment
	StatusNotApplicable Status = "NotApplicable"
)

// Result represents the result of a health check
type Result struct {
	// Status of the health check
	Status Status

	// Message provides a summary of the health check result
	Message string

	// Details contains additional information about the check
	Details string

	// Recommendations provides guidance on addressing any issues
	Recommendations []string

	// RawData contains any raw output data from the check
	RawData string

	// ExecutionTime is the time it took to run the check
	ExecutionTime time.Duration
}

// Check represents a health check
type Check interface {
	// Name returns the name of the health check
	Name() string

	// Description returns a description of what the health check does
	Description() string

	// Category returns the category the health check belongs to
	Category() string

	// Run executes the health check and returns the result
	Run() (Result, error)
}

// CategoryConfig contains configuration options for a category of health checks
type CategoryConfig struct {
	// Name of the category
	Name string

	// Description of the category
	Description string

	// Enabled indicates if the category is enabled
	Enabled bool
}

// Config contains configuration options for health checks
type Config struct {
	// Categories contains the configuration for each category
	Categories map[string]CategoryConfig

	// OutputDir is the directory where results will be written
	OutputDir string

	// VerboseOutput enables verbose output
	VerboseOutput bool

	// OnlyCategory runs only checks in the specified category
	OnlyCategory string

	// Timeout is the maximum time allowed for a check to run
	Timeout time.Duration
}

// BaseCheck provides a basic implementation of the Check interface
type BaseCheck struct {
	// ID is the unique identifier for the check
	ID string

	// NameStr is the name of the check
	NameStr string

	// DescriptionStr is the description of the check
	DescriptionStr string

	// CategoryStr is the category the check belongs to
	CategoryStr string
}

// Name returns the name of the health check
func (b *BaseCheck) Name() string {
	return b.NameStr
}

// Description returns a description of what the health check does
func (b *BaseCheck) Description() string {
	return b.DescriptionStr
}

// Category returns the category the health check belongs to
func (b *BaseCheck) Category() string {
	return b.CategoryStr
}

// NewResult creates a new Result with the given status and message
func NewResult(status Status, message string) Result {
	return Result{
		Status:          status,
		Message:         message,
		ExecutionTime:   0,
		Recommendations: []string{},
	}
}

// String returns a string representation of the result
func (r Result) String() string {
	return fmt.Sprintf("[%s] %s", r.Status, r.Message)
}

// AddRecommendation adds a recommendation to the result
func (r *Result) AddRecommendation(recommendation string) {
	r.Recommendations = append(r.Recommendations, recommendation)
}

// WithDetails adds details to the result and returns the result
func (r *Result) WithDetails(details string) *Result {
	r.Details = details
	return r
}

// WithRawData adds raw data to the result and returns the result
func (r *Result) WithRawData(rawData string) *Result {
	r.RawData = rawData
	return r
}

// WithExecutionTime sets the execution time for the result and returns the result
func (r *Result) WithExecutionTime(executionTime time.Duration) *Result {
	r.ExecutionTime = executionTime
	return r
}
