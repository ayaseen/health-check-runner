/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements the core runner for executing health checks. It:

- Manages the execution of multiple health checks
- Supports parallel or sequential execution modes
- Handles timeouts and error conditions
- Collects and organizes check results
- Provides progress reporting during check execution
- Implements filtering by category and other configuration options

This component orchestrates the execution of health checks and collects their results for reporting.
*/

package healthcheck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/schollz/progressbar/v3"
)

// Config defines the configuration for the runner
type Config struct {
	// OutputDir is where reports will be saved
	OutputDir string

	// CategoryFilter limits checks to specific categories
	CategoryFilter []types.Category

	// Timeout is the maximum time allowed for a check
	Timeout time.Duration

	// Parallel indicates whether checks should run in parallel
	Parallel bool

	// SkipProgressBar indicates whether to skip the progress bar
	SkipProgressBar bool

	// VerboseOutput enables verbose output
	VerboseOutput bool

	// FailFast stops execution when a critical error is encountered
	FailFast bool
}

// Runner executes health checks and collects results
type Runner struct {
	checks      []Check
	config      Config
	results     map[string]Result
	progressBar *progressbar.ProgressBar
	mu          sync.Mutex
}

// NewRunner creates a new health check runner
func NewRunner(config Config) *Runner {
	return &Runner{
		checks:  []Check{},
		config:  config,
		results: make(map[string]Result),
	}
}

// AddCheck adds a health check to the runner
func (r *Runner) AddCheck(check Check) {
	r.checks = append(r.checks, check)
}

// AddChecks adds multiple health checks to the runner
func (r *Runner) AddChecks(checks []Check) {
	for _, check := range checks {
		r.AddCheck(check)
	}
}

// GetChecks returns all registered health checks
func (r *Runner) GetChecks() []Check {
	return r.checks
}

// normalizeCategory maps legacy categories to their standardized versions
func normalizeCategory(category types.Category) types.Category {
	switch category {
	case "Infra", "Infrastructure":
		return types.CategoryClusterConfig
	case "Network":
		return types.CategoryNetworking
	case "Cluster":
		return types.CategoryClusterConfig
	case "App Dev":
		return types.CategoryApplications
	case "Monitoring":
		return types.CategoryOpReady
	default:
		return category
	}
}

// Run executes all registered health checks
func (r *Runner) Run() error {
	if len(r.checks) == 0 {
		return fmt.Errorf("no health checks registered")
	}

	// Filter checks by category if specified
	var checksToRun []Check
	if len(r.config.CategoryFilter) > 0 {
		for _, check := range r.checks {
			normalizedCheckCategory := normalizeCategory(check.Category())
			for _, cat := range r.config.CategoryFilter {
				normalizedFilterCategory := normalizeCategory(cat)
				if normalizedCheckCategory == normalizedFilterCategory {
					checksToRun = append(checksToRun, check)
					break
				}
			}
		}
	} else {
		checksToRun = r.checks
	}

	if len(checksToRun) == 0 {
		return fmt.Errorf("no health checks match the specified categories")
	}

	// Initialize progress bar if enabled
	if !r.config.SkipProgressBar {
		fmt.Println("OpenShift Health Check in Progress ...")

		// Create a simple progress bar without a description
		r.progressBar = progressbar.NewOptions(len(checksToRun),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerPadding: " ",
				BarStart:      "|",
				BarEnd:        "|",
			}),
		)
	}

	// Run checks in parallel or sequentially based on configuration
	if r.config.Parallel {
		r.runParallel(checksToRun)
	} else {
		r.runSequential(checksToRun)
	}

	return nil
}

// runSequential runs health checks sequentially
func (r *Runner) runSequential(checks []Check) {
	for _, check := range checks {
		// Create a context with timeout if configured
		ctx := context.Background()
		if r.config.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), r.config.Timeout)
			defer cancel()
		}

		// Run the check
		result, err := r.runCheck(ctx, check)

		// Store the result
		r.mu.Lock()
		r.results[check.ID()] = result
		r.mu.Unlock()

		// Print verbose output if enabled but no progress bar is used
		if r.config.VerboseOutput && r.config.SkipProgressBar {
			fmt.Printf("[%s] %s: %s\n", result.Status, check.Name(), result.Message)
		}

		// Increment progress bar if enabled
		if !r.config.SkipProgressBar && r.progressBar != nil {
			_ = r.progressBar.Add(1)
		}

		// Handle fail-fast if configured
		if r.config.FailFast && result.Status == types.StatusCritical && err != nil {
			break
		}
	}
}

// runParallel runs health checks in parallel
func (r *Runner) runParallel(checks []Check) {
	var wg sync.WaitGroup
	wg.Add(len(checks))

	// To track completed checks
	completedChecks := sync.Map{}

	// Update display mutex
	var displayMutex sync.Mutex

	for _, check := range checks {
		go func(c Check) {
			defer wg.Done()

			// Create a context with timeout if configured
			ctx := context.Background()
			if r.config.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), r.config.Timeout)
				defer cancel()
			}

			// Run the check
			result, _ := r.runCheck(ctx, c)

			// Store the result
			r.mu.Lock()
			r.results[c.ID()] = result
			r.mu.Unlock()

			// Print verbose output if enabled but no progress bar is used
			if r.config.VerboseOutput && r.config.SkipProgressBar {
				fmt.Printf("[%s] %s: %s\n", result.Status, c.Name(), result.Message)
			}

			// Mark check as completed
			completedChecks.Store(c.ID(), true)

			// Increment progress bar if enabled
			if !r.config.SkipProgressBar && r.progressBar != nil {
				displayMutex.Lock()
				// Count completed checks for progress bar
				completed := 0
				completedChecks.Range(func(key, value interface{}) bool {
					completed++
					return true
				})

				_ = r.progressBar.Set(completed)
				displayMutex.Unlock()
			}
		}(check)
	}

	wg.Wait()
}

// runCheck executes a single health check
func (r *Runner) runCheck(ctx context.Context, check Check) (Result, error) {
	// Track execution time
	startTime := time.Now()

	// Create a channel for the result
	resultCh := make(chan Result, 1)
	errCh := make(chan error, 1)

	// Run the check in a goroutine
	go func() {
		result, err := check.Run()
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Wait for the check to complete or timeout
	select {
	case result := <-resultCh:
		// Add execution time to the result
		result = result.WithExecutionTime(time.Since(startTime))
		return result, nil

	case err := <-errCh:
		result := NewResult(check.ID(), types.StatusCritical, fmt.Sprintf("Check failed: %v", err), types.ResultKeyRequired)
		result = result.WithExecutionTime(time.Since(startTime))
		return result, err

	case <-ctx.Done():
		result := NewResult(check.ID(), types.StatusCritical, "Check timed out", types.ResultKeyRequired)
		result = result.WithExecutionTime(time.Since(startTime))
		return result, ctx.Err()
	}
}

// GetResults returns all health check results
func (r *Runner) GetResults() map[string]Result {
	return r.results
}

// GetResultsByCategory returns health check results grouped by category
func (r *Runner) GetResultsByCategory() map[types.Category][]Result {
	resultsByCategory := make(map[types.Category][]Result)

	for _, check := range r.checks {
		if result, exists := r.results[check.ID()]; exists {
			// Normalize the category
			category := normalizeCategory(check.Category())
			resultsByCategory[category] = append(resultsByCategory[category], result)
		}
	}

	return resultsByCategory
}

// GetResultsByStatus returns health check results grouped by status
func (r *Runner) GetResultsByStatus() map[types.Status][]Result {
	resultsByStatus := make(map[types.Status][]Result)

	for _, result := range r.results {
		resultsByStatus[result.Status] = append(resultsByStatus[result.Status], result)
	}

	return resultsByStatus
}

// CountByStatus returns the count of results by status
func (r *Runner) CountByStatus() map[types.Status]int {
	counts := make(map[types.Status]int)

	for _, result := range r.results {
		counts[result.Status]++
	}

	return counts
}
