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

// LoggingInstallCheck checks if OpenShift Logging is installed and configured
type LoggingInstallCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingInstallCheck creates a new logging check
func NewLoggingInstallCheck() *LoggingInstallCheck {
	return &LoggingInstallCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-install",
			"OpenShift Logging Installation",
			"Checks if OpenShift Logging is installed and configured correctly",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingInstallCheck) Run() (healthcheck.Result, error) {
	// Check if the ClusterLogging CRD exists and "instance" is deployed
	out, err := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging")
	if err != nil {
		// If an error occurred, logging might not be installed
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Logging is not installed",
			types.ResultKeyRecommended,
		), nil
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed logging information"
	}

	// Check if "instance" is deployed and "Managed"
	isConfigured := strings.Contains(out, "instance") && strings.Contains(out, "Managed")

	if !isConfigured {
		// Get the OpenShift version for recommendations
		version, verErr := utils.GetOpenShiftMajorMinorVersion()
		if verErr != nil {
			version = "4.10" // Fallback version
		}

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Logging is not properly configured",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Deploy the logging subsystem to aggregate logs from your OpenShift Container Platform cluster")
		result.AddRecommendation(fmt.Sprintf("Follow the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/cluster-logging-deploying", version))

		result.Detail = detailedOut
		return result, nil
	}

	// Logging is installed and configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift Logging is installed and configured",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}

// LoggingHealthCheck checks if OpenShift Logging components are healthy
type LoggingHealthCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingHealthCheck creates a new logging health check
func NewLoggingHealthCheck() *LoggingHealthCheck {
	return &LoggingHealthCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-health",
			"OpenShift Logging Health",
			"Checks if OpenShift Logging components are functioning and healthy",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingHealthCheck) Run() (healthcheck.Result, error) {
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

	// Get the status of the ClusterLogging instance
	out, err := utils.RunCommand("oc", "get", "clusterlogging", "instance", "-n", "openshift-logging", "-o", "yaml")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get logging status",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting logging status: %v", err)
	}

	// Check if status is unhealthy (yellow or red)
	isUnhealthy := strings.Contains(out, "status: yellow") || strings.Contains(out, "status: red")

	if isUnhealthy {
		// Get the OpenShift version for recommendations
		version, verErr := utils.GetOpenShiftMajorMinorVersion()
		if verErr != nil {
			version = "4.10" // Fallback version
		}

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Logging is not healthy (status is yellow or red)",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Investigate the root cause of logging issues")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-cluster-status", version))

		result.Detail = out
		return result, nil
	}

	// Logging is healthy
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift Logging is healthy",
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}

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

// LoggingForwarderCheck checks if log forwarding is configured
type LoggingForwarderCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingForwarderCheck creates a new log forwarder check
func NewLoggingForwarderCheck() *LoggingForwarderCheck {
	return &LoggingForwarderCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-forwarder",
			"Log Forwarding",
			"Checks if log forwarding is configured for long-term storage",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingForwarderCheck) Run() (healthcheck.Result, error) {
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

	// Check for ClusterLogForwarder
	out, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging", "-o", "yaml")
	if err != nil {
		// ClusterLogForwarder might not exist
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Log forwarding is not configured",
			types.ResultKeyRecommended,
		), nil
	}

	// Check if it's properly configured with outputs, pipelines, etc.
	emptyConfig := strings.Contains(out, "outputs: []") || strings.Contains(out, "pipelines: []") || strings.Contains(out, "inputs: []")

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	if emptyConfig {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Log forwarding is configured but has empty pipelines or outputs",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure proper log forwarding for long-term storage")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = out
		return result, nil
	}

	// Log forwarding is properly configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Log forwarding is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}

// LoggingPlacementCheck checks if Elasticsearch pods are placed on appropriate nodes
type LoggingPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingPlacementCheck creates a new logging placement check
func NewLoggingPlacementCheck() *LoggingPlacementCheck {
	return &LoggingPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-placement",
			"Logging Component Placement",
			"Checks if Elasticsearch pods are scheduled on appropriate nodes",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingPlacementCheck) Run() (healthcheck.Result, error) {
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

	// Get Elasticsearch pods
	out, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch", "-o", "wide")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Elasticsearch pods",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Elasticsearch pods: %v", err)
	}

	// Get node information
	nodeOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "name")
	if err != nil || nodeOut == "" {
		// There are no infra nodes defined, so this check is not applicable
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No infrastructure nodes found in the cluster",
			types.ResultKeyRecommended,
		), nil
	}

	// Check if all pods are on infra nodes
	infraNodeNames := []string{}
	for _, line := range strings.Split(nodeOut, "\n") {
		if line != "" {
			// Extract node name from format like "node/node-name"
			parts := strings.Split(line, "/")
			if len(parts) > 1 {
				infraNodeNames = append(infraNodeNames, parts[1])
			}
		}
	}

	allOnInfraNodes := true
	podsOnNonInfraNodes := []string{}

	for _, line := range strings.Split(out, "\n") {
		if line != "" && !strings.HasPrefix(line, "NAME") { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 7 { // Ensure we have enough fields to access node name
				podName := fields[0]
				nodeName := fields[6]

				onInfraNode := false
				for _, infraNode := range infraNodeNames {
					if nodeName == infraNode {
						onInfraNode = true
						break
					}
				}

				if !onInfraNode {
					allOnInfraNodes = false
					podsOnNonInfraNodes = append(podsOnNonInfraNodes, fmt.Sprintf("%s on %s", podName, nodeName))
				}
			}
		}
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	if !allOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Some Elasticsearch pods are not scheduled on infrastructure nodes",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Move Elasticsearch pods to infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#infrastructure-moving-logging_cluster-logging-moving", version))

		detail := fmt.Sprintf("Elasticsearch pods not on infrastructure nodes:\n%s\n\nInfrastructure nodes:\n%s\n\nPod details:\n%s",
			strings.Join(podsOnNonInfraNodes, "\n"),
			nodeOut,
			out)
		result.Detail = detail
		return result, nil
	}

	// All pods are on infra nodes
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"All Elasticsearch pods are scheduled on infrastructure nodes",
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}

// Helper function to check if OpenShift Logging is installed
func isLoggingInstalled() (bool, error) {
	// Check if the ClusterLogging CRD exists
	_, err := utils.RunCommand("oc", "get", "crd", "clusterloggings.logging.openshift.io")
	if err != nil {
		// The CRD doesn't exist, logging is not installed
		return false, nil
	}

	// Check if the namespace exists
	_, err = utils.RunCommand("oc", "get", "namespace", "openshift-logging")
	if err != nil {
		// The namespace doesn't exist, logging is not installed
		return false, nil
	}

	// Check if there's an instance deployed
	out, err := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging")
	if err != nil {
		// No ClusterLogging instance found
		return false, nil
	}

	// Check if "instance" is deployed
	return strings.Contains(out, "instance"), nil
}

// Helper function to extract disk usage from Elasticsearch output
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

// Helper function to extract message from condition
func extractMessage(condition string) string {
	re := regexp.MustCompile(`(?m)message:\s+(.*)`)
	match := re.FindStringSubmatch(condition)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

// Helper function to extract disk usage percentage
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
