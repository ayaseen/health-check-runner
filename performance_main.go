/*
This file provides a standalone test for the performance health checks.

To run this test:
1. Save this file in the root of your project directory
2. Run: go run performance_main.go
3. Analyze the output to verify the performance checks work as expected

Requirements:
- Must be run from a machine with access to an OpenShift cluster
- You must be logged in to the cluster with 'oc login' before running
- Your user must have sufficient permissions to access metrics and node information
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	// Depending on your module name, adjust these imports
	"github.com/ayaseen/health-check-runner/pkg/checks/performance"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// MockReporter is a simplified reporter for testing
type MockReporter struct {
	results map[string]types.Result
}

// PrintResults prints the health check results
func (r *MockReporter) PrintResults() {
	// Print a simple header
	fmt.Println("==== Performance Health Check Results ====")
	fmt.Println()

	// Print each result
	for id, result := range r.results {
		fmt.Printf("Check ID: %s\n", id)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Message: %s\n", result.Message)
		fmt.Println("Recommendations:")
		for _, rec := range result.Recommendations {
			fmt.Printf("- %s\n", rec)
		}
		fmt.Println()
		fmt.Println("Details:")
		fmt.Println(result.Detail)
		fmt.Println("====================================")
		fmt.Println()
	}
}

// ExportResultsToJSON exports the results to a JSON file
func (r *MockReporter) ExportResultsToJSON(filename string) error {
	// Create a serializable structure
	type exportResult struct {
		CheckID         string   `json:"check_id"`
		Status          string   `json:"status"`
		Message         string   `json:"message"`
		Recommendations []string `json:"recommendations"`
		Detail          string   `json:"detail"`
		ExecutionTime   string   `json:"execution_time"`
	}

	var results []exportResult
	for id, result := range r.results {
		results = append(results, exportResult{
			CheckID:         id,
			Status:          string(result.Status),
			Message:         result.Message,
			Recommendations: result.Recommendations,
			Detail:          result.Detail,
			ExecutionTime:   result.ExecutionTime,
		})
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results to JSON: %v", err)
	}

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %v", err)
	}

	return nil
}

// ExportResultsToAsciiDoc exports the results to an AsciiDoc file
func (r *MockReporter) ExportResultsToAsciiDoc(filename string) error {
	var output strings.Builder

	// Write header
	output.WriteString("= OpenShift Performance Health Check Results\n\n")
	output.WriteString("== Summary\n\n")

	// Create summary table
	output.WriteString("[cols=\"1,2,1,2\", options=\"header\"]\n|===\n")
	output.WriteString("|Check ID|Status|Message|Recommendations\n\n")

	for id, result := range r.results {
		// Format recommendations
		recs := ""
		for _, rec := range result.Recommendations {
			recs += "* " + rec + "\n"
		}

		output.WriteString(fmt.Sprintf("|%s|%s|%s|%s\n",
			id, string(result.Status), result.Message, recs))
	}
	output.WriteString("|===\n\n")

	// Add details for each check
	for id, result := range r.results {
		output.WriteString(fmt.Sprintf("== %s\n\n", id))
		output.WriteString(fmt.Sprintf("*Status*: %s\n\n", string(result.Status)))
		output.WriteString(fmt.Sprintf("*Message*: %s\n\n", result.Message))

		output.WriteString("*Recommendations*:\n\n")
		for _, rec := range result.Recommendations {
			output.WriteString("* " + rec + "\n")
		}
		output.WriteString("\n")

		output.WriteString("*Details*:\n\n")
		output.WriteString(result.Detail)
		output.WriteString("\n\n")
	}

	// Write to file
	if err := os.WriteFile(filename, []byte(output.String()), 0644); err != nil {
		return fmt.Errorf("failed to write results to file: %v", err)
	}

	return nil
}

// Test if OpenShift is accessible
func testOpenShiftAccess() bool {
	cmd := "oc whoami"
	_, err := utils.RunCommand("bash", "-c", cmd)
	return err == nil
}

func main() {
	fmt.Println("OpenShift Performance Health Check Tester")
	fmt.Println("=========================================")
	fmt.Println()

	// Test OpenShift access
	fmt.Print("Testing OpenShift access... ")
	if !testOpenShiftAccess() {
		fmt.Println("FAILED")
		fmt.Println("Error: Could not access OpenShift. Please make sure you are logged in with 'oc login'.")
		os.Exit(1)
	}
	fmt.Println("OK")

	// Create and run the performance checks
	fmt.Println("Running performance health checks...")

	// Create a new performance check
	perfCheck := performance.NewClusterPerformanceCheck()

	// Record start time
	startTime := time.Now()

	// Run the check
	result, err := perfCheck.Run()
	if err != nil {
		fmt.Printf("Error running performance check: %v\n", err)
		os.Exit(1)
	}

	// Calculate execution time
	executionTime := time.Since(startTime)
	executionTimeStr := executionTime.String()

	// Convert result to types.Result
	typesResult := types.Result{
		CheckID:         result.CheckID,
		Status:          result.Status,
		Message:         result.Message,
		ResultKey:       result.ResultKey,
		Detail:          result.Detail,
		Recommendations: result.Recommendations,
		ExecutionTime:   executionTimeStr,
		Metadata:        result.Metadata,
	}

	// Create a mock reporter and store the result
	reporter := &MockReporter{
		results: map[string]types.Result{
			perfCheck.ID(): typesResult,
		},
	}

	// Print results to console
	reporter.PrintResults()

	// Export results to files
	jsonFile := "performance_check_results.json"
	asciiDocFile := "performance_check_results.adoc"

	fmt.Printf("Exporting results to JSON file: %s\n", jsonFile)
	if err := reporter.ExportResultsToJSON(jsonFile); err != nil {
		fmt.Printf("Error exporting to JSON: %v\n", err)
	}

	fmt.Printf("Exporting results to AsciiDoc file: %s\n", asciiDocFile)
	if err := reporter.ExportResultsToAsciiDoc(asciiDocFile); err != nil {
		fmt.Printf("Error exporting to AsciiDoc: %v\n", err)
	}

	fmt.Println("\nTest completed successfully!")
	fmt.Printf("Performance check execution time: %s\n", executionTimeStr)
}
