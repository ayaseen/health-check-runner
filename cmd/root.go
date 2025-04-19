/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements the root command for the health check CLI application. It handles:

- Command-line flag parsing for configuration options
- Setting up the health check runner with the appropriate configuration
- Executing health checks based on the selected type (OpenShift, application, or all)
- Generating and compressing reports in various formats (AsciiDoc, HTML, JSON, summary)
- Validating user inputs to ensure proper configuration

The root command serves as the main entry point for the health check utility, orchestrating the entire process from configuration to report generation.
*/

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/checks"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	// Config options
	checkType       string
	outputDir       string
	reportFormat    string
	includeDetails  bool
	parallel        bool
	timeout         int
	skipProgressBar bool
	categoryFilter  []string
	verboseOutput   bool
	failFast        bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hc-runner",
	Short: "Performs a health check against OpenShift clusters",
	Long: `This application helps perform health checks against OpenShift clusters.
The application runs a variety of checks and generates a formatted report with the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flags
		if err := validateFlags(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create directory for output
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
			os.Exit(1)
		}

		// Parse category filters
		var categories []types.Category
		if len(categoryFilter) > 0 {
			for _, cat := range categoryFilter {
				categories = append(categories, types.Category(cat))
			}
		}

		// Create runner configuration
		config := healthcheck.Config{
			OutputDir:       outputDir,
			CategoryFilter:  categories,
			Timeout:         time.Duration(timeout) * time.Second,
			Parallel:        parallel,
			SkipProgressBar: skipProgressBar,
			VerboseOutput:   verboseOutput,
			FailFast:        failFast,
		}

		// Create runner
		runner := healthcheck.NewRunner(config)

		// Register checks based on the check type
		switch checkType {
		case "openshift":
			runner.AddChecks(checks.GetOpenShiftChecks())
		case "application":
			runner.AddChecks(checks.GetApplicationChecks())
		case "all":
			runner.AddChecks(checks.GetAllChecks())
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown check type: %s\n", checkType)
			os.Exit(1)
		}

		// Run the checks
		if err := runner.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running health checks: %v\n", err)
			os.Exit(1)
		}

		// Create reporter configuration
		reporterConfig := healthcheck.ReportConfig{
			Format:                 types.ReportFormat(reportFormat),
			OutputDir:              outputDir,
			Filename:               "health-check-report",
			IncludeTimestamp:       true,
			IncludeDetailedResults: includeDetails,
			Title:                  "OpenShift Health Check Report",
			GroupByCategory:        true,
			ColorOutput:            true,
			UseEnhancedAsciiDoc:    true,
		}

		// Create reporter
		reporter := healthcheck.NewReporter(reporterConfig, runner)

		// Set the enhanced AsciiDoc generator to avoid circular imports
		reporter.SetEnhancedAsciiDocGenerator(utils.GenerateEnhancedAsciiDocReport)

		// Generate report
		reportPath, err := reporter.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}

		// Password for ZIP encryption
		const password = "7e5eed48001f9a407bbb87b29c32871b"

		// Compress the report with password protection
		zipPath, err := utils.CompressWithPassword(reportPath, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to compress report: %v\n", err)
			fmt.Printf("\nReport generated at: %s\n", reportPath)
		} else {
			fmt.Printf("\nCompressed report generated at: %s", zipPath)

			// Delete the original report
			if err := os.Remove(reportPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to delete original report: %v\n", err)
			}
		}
	},
}

// Modified Execute function to skip OpenShift connectivity check for exec-summary command
func Execute() {
	// Check if the command is exec-summary
	// If exec-summary, skip OpenShift connectivity check
	isExecSummary := false
	for _, arg := range os.Args {
		if arg == "exec-summary" {
			isExecSummary = true
			break
		}
	}

	// Verify OpenShift connection before running health checks, but not for exec-summary command
	if !isExecSummary {
		verifyOpenShiftConnection()
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Define flags
	rootCmd.PersistentFlags().StringVar(&checkType, "check", "all", "Type of health check to run (openshift, application, all)")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", "resources", "Directory where reports will be saved")
	rootCmd.PersistentFlags().StringVar(&reportFormat, "format", "asciidoc", "Report format (asciidoc, html, json, summary)")
	rootCmd.PersistentFlags().BoolVar(&includeDetails, "details", true, "Include detailed results in the report")
	rootCmd.PersistentFlags().BoolVar(&parallel, "parallel", false, "Run checks in parallel")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 300, "Timeout for health checks in seconds (0 for no timeout)")
	rootCmd.PersistentFlags().BoolVar(&skipProgressBar, "no-progress", false, "Disable progress bar")
	rootCmd.PersistentFlags().StringSliceVar(&categoryFilter, "category", []string{}, "Run only checks in specified categories (comma-separated)")
	rootCmd.PersistentFlags().BoolVar(&verboseOutput, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&failFast, "fail-fast", false, "Stop on first critical failure")
}

// validateFlags validates the command line flags with improved validation
func validateFlags() error {
	// Validate check type
	validCheckTypes := map[string]bool{
		"openshift":   true,
		"application": true,
		"storage":     true,
		"all":         true,
	}

	if !validCheckTypes[checkType] {
		return fmt.Errorf("invalid check type: %s (must be one of: openshift, application, storage, all)", checkType)
	}

	// Validate report format
	validFormats := map[string]bool{
		"asciidoc": true,
		"html":     true,
		"json":     true,
		"summary":  true,
	}

	if !validFormats[reportFormat] {
		return fmt.Errorf("invalid report format: %s (must be one of: asciidoc, html, json, summary)", reportFormat)
	}

	// Validate timeout
	if timeout < 0 {
		return fmt.Errorf("timeout must be greater than or equal to 0")
	}

	// Validate category filters - using the updated category names
	validCategories := map[string]bool{
		"Networking":     true,
		"Applications":   true,
		"Op-Ready":       true,
		"Security":       true,
		"Cluster Config": true,
		"Storage":        true,
		"Performance":    true,
	}

	for _, category := range categoryFilter {
		if !validCategories[category] {
			return fmt.Errorf("invalid category: %s (must be one of: Networking, Applications, Op-Ready, Security, Cluster Config, Storage)", category)
		}
	}

	// Validate output directory
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	return nil
}

// verifyOpenShiftConnection checks if the OpenShift API is accessible and exits with an error message if not
func verifyOpenShiftConnection() {
	accessible, message := utils.VerifyOpenShiftAccess()
	if !accessible {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		os.Exit(1)
	}
	fmt.Println(message)
}
