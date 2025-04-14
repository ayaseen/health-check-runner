package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
