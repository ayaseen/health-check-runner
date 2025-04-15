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

// LoggingStorageCheck checks if logging has sufficient storage space
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
			"Checks if logging components have sufficient storage space",
			types.CategoryOpReady,
		),
		warningThreshold:  85,
		criticalThreshold: 95,
	}
}

// Run executes the health check
func (c *LoggingStorageCheck) Run() (healthcheck.Result, error) {
	// Detect logging configuration
	loggingInfo, err := DetectLoggingConfiguration()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to detect logging configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error detecting logging configuration: %v", err)
	}

	// If logging is not installed, return NotApplicable
	if loggingInfo.Type == LoggingTypeNone {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Logging is not installed",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.14" // Update to a more recent default version
	}

	// Check storage based on logging type
	if loggingInfo.Type == LoggingTypeLoki {
		return checkLokiStorage(c, version)
	} else {
		return checkElasticsearchStorage(c, version)
	}
}

// checkLokiStorage checks Loki storage usage
func checkLokiStorage(c *LoggingStorageCheck, version string) (healthcheck.Result, error) {
	// First check if S3 storage is used by examining the LokiStack CR
	lokiStackOut, err := utils.RunCommand("oc", "get", "lokistack", "-n", "openshift-logging", "-o", "yaml")

	// Check if using S3 or similar object storage
	isObjectStorage := strings.Contains(lokiStackOut, "type: s3") ||
		strings.Contains(lokiStackOut, "type: gcs") ||
		strings.Contains(lokiStackOut, "type: azure")

	if isObjectStorage {
		// For object storage, we should check that everything is configured properly
		// rather than checking for disk space

		// Check Loki component status
		if strings.Contains(lokiStackOut, "type: Warning") &&
			strings.Contains(lokiStackOut, "StorageNeedsSchemaUpdate") {

			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"Loki storage schema needs to be updated",
				types.ResultKeyRecommended,
			)

			result.AddRecommendation("Update the Loki storage schema to the latest version")
			result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

			result.Detail = lokiStackOut
			return result, nil
		}

		// Check if all components are ready
		if !strings.Contains(lokiStackOut, "reason: ReadyComponents") ||
			!strings.Contains(lokiStackOut, "status: 'True'") ||
			!strings.Contains(lokiStackOut, "type: Ready") {

			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"Some Loki components may not be ready - check object storage configuration",
				types.ResultKeyRecommended,
			)

			result.AddRecommendation("Verify S3 endpoint is accessible and credentials are correct")
			result.AddRecommendation("Check object storage bucket permissions")
			result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

			result.Detail = lokiStackOut
			return result, nil
		}

		// All checks passed
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Loki is properly configured with object storage",
			types.ResultKeyNoChange,
		)

		result.Detail = lokiStackOut
		return result, nil
	}

	// If not using object storage, continue with the original PVC-based checks
	// Get Loki PVC information
	pvcOut, err := utils.RunCommand("oc", "get", "pvc", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=loki")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Failed to get Loki storage information",
			types.ResultKeyRecommended,
		), nil
	}

	// Check if Loki PVCs exist
	if !strings.Contains(pvcOut, "loki") {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No Loki storage PVCs found",
			types.ResultKeyRecommended,
		), nil
	}

	// Get detailed PVC information
	// This could be used in the future for more detailed analysis
	_, _ = utils.RunCommand("oc", "get", "pvc", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=loki", "-o", "yaml")

	// Get pod disk usage information
	diskUsageCmd := "oc exec $(oc get pods -n openshift-logging -l app.kubernetes.io/component=loki -o name | head -1) -n openshift-logging -- df -h /var/loki"
	diskUsageOut, err := utils.RunCommand("bash", "-c", diskUsageCmd)

	// Parse disk usage percentage if available
	diskUsage := -1
	if err == nil {
		re := regexp.MustCompile(`(\d+)%`)
		match := re.FindStringSubmatch(diskUsageOut)
		if len(match) == 2 {
			diskUsage, _ = strconv.Atoi(match[1])
		}
	}

	// If we couldn't determine disk usage through exec, check PVC utilization
	if diskUsage == -1 {
		// Could implement an alternative check here
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Could not determine Loki storage usage",
			types.ResultKeyAdvisory,
		)
		result.Detail = fmt.Sprintf("PVC Information:\n%s\n\nDisk Usage Command Result:\n%s", pvcOut, diskUsageOut)
		return result, nil
	}

	// Check disk usage against thresholds
	if diskUsage >= c.criticalThreshold {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			fmt.Sprintf("Loki disk usage is critical: %d%%", diskUsage),
			types.ResultKeyRequired,
		)

		result.AddRecommendation("Expand the available storage for Loki")
		result.AddRecommendation("Reduce the log retention period")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

		result.Detail = fmt.Sprintf("PVC Information:\n%s\n\nDisk Usage:\n%s", pvcOut, diskUsageOut)
		return result, nil
	} else if diskUsage >= c.warningThreshold {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Loki disk usage is high: %d%%", diskUsage),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Consider expanding the available storage for Loki")
		result.AddRecommendation("Consider reducing the log retention period")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

		result.Detail = fmt.Sprintf("PVC Information:\n%s\n\nDisk Usage:\n%s", pvcOut, diskUsageOut)
		return result, nil
	}

	// Storage usage is normal
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Loki disk usage is normal: %d%%", diskUsage),
		types.ResultKeyNoChange,
	)
	result.Detail = fmt.Sprintf("PVC Information:\n%s\n\nDisk Usage:\n%s", pvcOut, diskUsageOut)
	return result, nil
}

// checkElasticsearchStorage checks Elasticsearch storage usage
func checkElasticsearchStorage(c *LoggingStorageCheck, version string) (healthcheck.Result, error) {
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
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-elasticsearch-storage_cluster-logging-elasticsearch", version))

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
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-elasticsearch-storage_cluster-logging-elasticsearch", version))

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
