package healthcheck

import (
	"time"
)

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
	// CategoryCluster is for cluster-level checks
	CategoryCluster Category = "Cluster"

	// CategorySecurity is for security-related checks
	CategorySecurity Category = "Security"

	// CategoryNetworking is for networking-related checks
	CategoryNetworking Category = "Networking"

	// CategoryStorage is for storage-related checks
	CategoryStorage Category = "Storage"

	// CategoryApplications is for application-related checks
	CategoryApplications Category = "Applications"

	// CategoryMonitoring is for monitoring-related checks
	CategoryMonitoring Category = "Monitoring"

	// CategoryInfrastructure is for infrastructure-related checks
	CategoryInfrastructure Category = "Infrastructure"
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
	ExecutionTime time.Duration

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

	// Run executes the health check and returns the result
	Run() (Result, error)
}

// BaseCheck provides a basic implementation of a health check
type BaseCheck struct {
	// id is the unique identifier for the check
	id string

	// name is the human-readable name for the check
	name string

	// description is what the check does
	description string

	// category is the category the check belongs to
	category Category
}

// ID returns the unique identifier for the check
func (b *BaseCheck) ID() string {
	return b.id
}

// Name returns the human-readable name for the check
func (b *BaseCheck) Name() string {
	return b.name
}

// Description returns a description of what the check does
func (b *BaseCheck) Description() string {
	return b.description
}

// Category returns the category the check belongs to
func (b *BaseCheck) Category() Category {
	return b.category
}

// NewBaseCheck creates a new BaseCheck
func NewBaseCheck(id, name, description string, category Category) BaseCheck {
	return BaseCheck{
		id:          id,
		name:        name,
		description: description,
		category:    category,
	}
}

// NewResult creates a new Result
func NewResult(checkID string, status Status, message string, resultKey ResultKey) Result {
	return Result{
		CheckID:         checkID,
		Status:          status,
		Message:         message,
		ResultKey:       resultKey,
		Recommendations: []string{},
		Metadata:        make(map[string]string),
	}
}

// AddRecommendation adds a recommendation to the result
func (r *Result) AddRecommendation(recommendation string) {
	r.Recommendations = append(r.Recommendations, recommendation)
}

// AddMetadata adds or updates metadata in the result
func (r *Result) AddMetadata(key, value string) {
	r.Metadata[key] = value
}

// WithDetail adds detailed information to the result
func (r *Result) WithDetail(detail string) Result {
	r.Detail = detail
	return *r
}

// WithExecutionTime sets the execution time for the result
func (r *Result) WithExecutionTime(duration time.Duration) Result {
	r.ExecutionTime = duration
	return *r
}
