/*
Author: Amjad Yaseen, Updated by the refactoring team
Email: ayaseen@redhat.com
Date: 2023-03-06, Updated: 2025-04-12

This application performs health checks on OpenShift to provide visibility into various functionalities. It verifies the following aspects:

- OpenShift configurations: Verify OpenShift configuration meets the standard and best practices.
- Security: It examines the security measures in place, such as authentication and authorization configurations.
- Application Probes: It tests the health and readiness probes of deployed applications to ensure they are functioning correctly.
- Resource Usage: It monitors resource consumption of OpenShift clusters, including CPU, memory, and storage.

The purpose of this application is to provide administrators and developers with an overview of OpenShift's health and functionality, helping them identify potential issues and ensure the smooth operation of their OpenShift environment.
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
		}

		// Create reporter
		reporter := healthcheck.NewReporter(reporterConfig, runner)

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
			fmt.Printf("\nCompressed report generated at: %s\n", zipPath)
			fmt.Printf("Password: %s\n", password)

			// Delete the original report
			if err := os.Remove(reportPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to delete original report: %v\n", err)
			}
		}

		// Print summary
		printSummary(runner)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
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

	// Validate category filters
	validCategories := map[string]bool{
		"Cluster":        true,
		"Security":       true,
		"Networking":     true,
		"Storage":        true,
		"Applications":   true,
		"Monitoring":     true,
		"Infrastructure": true,
	}

	for _, category := range categoryFilter {
		if !validCategories[category] {
			return fmt.Errorf("invalid category: %s (must be one of: Cluster, Security, Networking, Storage, Applications, Monitoring, Infrastructure)", category)
		}
	}

	// Validate output directory
	if outputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	return nil
}

// printSummary prints a summary of the health check results with improved formatting
func printSummary(runner *healthcheck.Runner) {
	counts := runner.CountByStatus()

	fmt.Println("\nHealth Check Summary:")
	fmt.Println("---------------------")

	totalChecks := 0
	for _, count := range counts {
		totalChecks += count
	}

	fmt.Printf("Total checks: %d\n", totalChecks)

	// Print statuses in a consistent order
	statuses := []types.Status{
		types.StatusOK,
		types.StatusWarning,
		types.StatusCritical,
		types.StatusUnknown,
		types.StatusNotApplicable,
	}

	for _, status := range statuses {
		count, exists := counts[status]
		if exists {
			// Add color to the output if it's a terminal
			switch status {
			case types.StatusOK:
				fmt.Printf("\033[32mOK\033[0m: %d\n", count)
			case types.StatusWarning:
				fmt.Printf("\033[33mWarning\033[0m: %d\n", count)
			case types.StatusCritical:
				fmt.Printf("\033[31mCritical\033[0m: %d\n", count)
			case types.StatusUnknown:
				fmt.Printf("\033[37mUnknown\033[0m: %d\n", count)
			case types.StatusNotApplicable:
				fmt.Printf("Not Applicable: %d\n", count)
			}
		}
	}

	// Check for issues to report
	warningResults := runner.GetResultsByStatus()[types.StatusWarning]
	criticalResults := runner.GetResultsByStatus()[types.StatusCritical]

	if len(warningResults) > 0 || len(criticalResults) > 0 {
		fmt.Println("\nIssues found:")

		// Print critical issues first
		if len(criticalResults) > 0 {
			fmt.Println("\nCritical issues:")
			for _, result := range criticalResults {
				// Find the check name
				var checkName string
				for _, check := range runner.GetChecks() {
					if check.ID() == result.CheckID {
						checkName = check.Name()
						break
					}
				}
				fmt.Printf("\033[31m[Critical]\033[0m %s: %s\n", checkName, result.Message)

				// Print recommendations for critical issues
				if len(result.Recommendations) > 0 {
					fmt.Println("  Recommendations:")
					for _, rec := range result.Recommendations {
						fmt.Printf("  - %s\n", rec)
					}
				}
			}
		}

		// Then print warnings
		if len(warningResults) > 0 {
			fmt.Println("\nWarnings:")
			for _, result := range warningResults {
				// Find the check name
				var checkName string
				for _, check := range runner.GetChecks() {
					if check.ID() == result.CheckID {
						checkName = check.Name()
						break
					}
				}
				fmt.Printf("\033[33m[Warning]\033[0m %s: %s\n", checkName, result.Message)

				// Print recommendations for warning issues
				if len(result.Recommendations) > 0 {
					fmt.Println("  Recommendations:")
					for _, rec := range result.Recommendations {
						fmt.Printf("  - %s\n", rec)
					}
				}
			}
		}
	} else {
		fmt.Println("\n\033[32mNo issues found.\033[0m")
	}

	// Print report location
	fmt.Printf("\nFor more details, refer to the generated report.\n")
}
