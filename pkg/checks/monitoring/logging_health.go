package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

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
