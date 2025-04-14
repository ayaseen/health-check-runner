package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"regexp"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingStorageCheck checks if Elasticsearch has sufficient storage space
type LoggingStorageCheck struct {
	healthcheck.BaseCheck
	warningThreshold  int
	criticalThreshold int
}

// NewLoggingStorageCheck creates a new logging storage check
func NewLoggingStorageCheck() *LoggingStorageCheck {
	return &LoggingStorageCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-storage",
			"OpenShift Logging Storage",
			"Checks if Elasticsearch has sufficient storage space",
			types.CategoryOpReady,
		),
		warningThreshold:  85,
		criticalThreshold: 95,
	}
}

// Run executes the health check
func (c *LoggingStorageCheck) Run() (healthcheck.Result, error) {
	// First check if logging is installed
	isInstalled, err := isLoggingInstalled()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to check OpenShift Logging installation",
			types.ResultKeyRequired,
		), fmt.Errorf("error checking logging installation: %v", err)
	}

	// If logging is not installed, return NotApplicable
	if !isInstalled {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Logging is not installed",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get Elasticsearch resource
	out, err := utils.RunCommand("oc", "get", "Elasticsearch", "elasticsearch", "-n", "openshift-logging", "-o", "yaml")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Elasticsearch information",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Elasticsearch information: %v", err)
	}

	// Extract disk usage from conditions
	diskUsage := getDiskStorageUsage(out)

	// If disk usage couldn't be determined
	if diskUsage == -1 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Could not determine Elasticsearch storage usage",
			types.ResultKeyAdvisory,
		)
		result.Detail = out
		return result, nil
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	// Check disk usage against thresholds
	if diskUsage >= c.criticalThreshold {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			fmt.Sprintf("Elasticsearch disk usage is critical: %d%%", diskUsage),
			types.ResultKeyRequired,
		)

		result.AddRecommendation("Expand the available storage for Elasticsearch")
		result.AddRecommendation("Reduce the log retention period")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = out
		return result, nil
	} else if diskUsage >= c.warningThreshold {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Elasticsearch disk usage is high: %d%%", diskUsage),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Consider expanding the available storage for Elasticsearch")
		result.AddRecommendation("Consider reducing the log retention period")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = out
		return result, nil
	}

	// Storage usage is normal
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Elasticsearch disk usage is normal: %d%%", diskUsage),
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}

// getDiskStorageUsage extracts disk usage from Elasticsearch output
func getDiskStorageUsage(output string) int {
	conditions := strings.SplitAfter(output, "- conditions:")
	for _, condition := range conditions {
		if strings.Contains(condition, "type: NodeStorage") && strings.Contains(condition, "status: \"True\"") {
			message := extractMessage(condition)
			return extractDiskUsage(message)
		}
	}
	return -1
}

// extractMessage extracts message from condition
func extractMessage(condition string) string {
	re := regexp.MustCompile(`(?m)message:\s+(.*)`)
	match := re.FindStringSubmatch(condition)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

// extractDiskUsage extracts disk usage percentage
func extractDiskUsage(message string) int {
	re := regexp.MustCompile(`\((.*?)\)`)
	match := re.FindStringSubmatch(message)
	if len(match) == 2 {
		diskUsageStr := match[1]
		diskUsageStr = strings.TrimSuffix(diskUsageStr, "%")
		diskUsageFloat, err := strconv.ParseFloat(diskUsageStr, 64)
		if err != nil {
			return -1
		}
		return int(diskUsageFloat)
	}
	return -1
}
