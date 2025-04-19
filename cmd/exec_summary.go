// cmd/exec_summary.go

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ayaseen/health-check-runner/pkg/utils/executive_summary"
	"github.com/spf13/cobra"
)

var (
	execSummaryClusterName  string
	execSummaryCustomerName string
	execSummaryReportPath   string
	execSummaryOutputDir    string
)

// execSummaryCmd represents the command for generating an executive summary
var execSummaryCmd = &cobra.Command{
	Use:   "exec-summary",
	Short: "Generate an executive summary from a health check report",
	Long: `This command generates an executive summary from a previously 
generated health check report, providing a high-level overview of the results.

The summary includes scores for each category of health checks, a list of items
requiring attention, and recommendations for improvement.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate flags
		if err := validateExecSummaryFlags(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Generate the executive summary
		if err := executive_summary.GenerateExecutiveSummary(execSummaryReportPath, execSummaryClusterName, execSummaryCustomerName, execSummaryOutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating executive summary: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(execSummaryCmd)

	// Define flags for the exec-summary command
	execSummaryCmd.Flags().StringVar(&execSummaryClusterName, "cluster", "UNKNOWN", "Cluster name (e.g., UAT, PROD)")
	execSummaryCmd.Flags().StringVar(&execSummaryCustomerName, "customer", "Customer", "Customer name")
	execSummaryCmd.Flags().StringVar(&execSummaryReportPath, "report", "", "Path to the health check report file (.adoc)")
	execSummaryCmd.Flags().StringVar(&execSummaryOutputDir, "output-dir", "resources", "Directory where the executive summary will be saved")

	// Mark required flags
	execSummaryCmd.MarkFlagRequired("report")
}

// validateExecSummaryFlags validates the command line flags
func validateExecSummaryFlags() error {
	// Check if report path is provided
	if execSummaryReportPath == "" {
		return fmt.Errorf("report path is required")
	}

	// Check if report file exists
	if _, err := os.Stat(execSummaryReportPath); os.IsNotExist(err) {
		return fmt.Errorf("report file not found: %s", execSummaryReportPath)
	}

	// Check if report file is an AsciiDoc file
	ext := filepath.Ext(execSummaryReportPath)
	if ext != ".adoc" {
		return fmt.Errorf("report file must be an AsciiDoc file (.adoc)")
	}

	return nil
}
