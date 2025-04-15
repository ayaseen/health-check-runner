/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file defines the core interfaces and structures for health checks. It includes:

- The Check interface that all health checks must implement
- BaseCheck structure providing common functionality for all checks
- Result structure for storing and managing check results
- Methods for managing check metadata, recommendations, and execution details
- Conversion utilities between internal and external result representations

This file forms the foundation of the health check framework, defining how checks are structured and how results are processed.
*/

package healthcheck

import (
	"time"

	"github.com/ayaseen/health-check-runner/pkg/types"
)

// BaseCheck provides a basic implementation of a health check
type BaseCheck struct {
	// id is the unique identifier for the check
	id string

	// name is the human-readable name for the check
	name string

	// description is what the check does
	description string

	// category is the category the check belongs to
	category types.Category
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
func (b *BaseCheck) Category() types.Category {
	return b.category
}

// NewBaseCheck creates a new BaseCheck
func NewBaseCheck(id, name, description string, category types.Category) BaseCheck {
	return BaseCheck{
		id:          id,
		name:        name,
		description: description,
		category:    category,
	}
}

// Result represents the result of a health check with execution time as duration
type Result struct {
	// CheckID is the unique identifier of the health check
	CheckID string

	// Status indicates the result status (OK, Warning, Critical, etc.)
	Status types.Status

	// Message is a brief description of the result
	Message string

	// ResultKey indicates the importance of the result in a report
	ResultKey types.ResultKey

	// Detail provides detailed information about the result
	Detail string

	// Recommendations are suggestions to address any issues
	Recommendations []string

	// ExecutionTime is how long the check took to run
	ExecutionTime time.Duration

	// Metadata is additional contextual information
	Metadata map[string]string
}

// NewResult creates a new Result
func NewResult(checkID string, status types.Status, message string, resultKey types.ResultKey) Result {
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
	result := *r // Create a copy
	result.Detail = detail
	return result
}

// WithExecutionTime sets the execution time for the result
func (r *Result) WithExecutionTime(duration time.Duration) Result {
	result := *r // Create a copy
	result.ExecutionTime = duration
	return result
}

// ToTypesResult converts a Result to types.Result
func (r *Result) ToTypesResult() types.Result {
	return types.Result{
		CheckID:         r.CheckID,
		Status:          r.Status,
		Message:         r.Message,
		ResultKey:       r.ResultKey,
		Detail:          r.Detail,
		Recommendations: r.Recommendations,
		ExecutionTime:   r.ExecutionTime.String(),
		Metadata:        r.Metadata,
	}
}

// Check defines the interface for a health check
type Check interface {
	types.Check

	// Run executes the health check and returns the result
	Run() (Result, error)
}
