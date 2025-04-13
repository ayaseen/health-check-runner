package cluster

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// InfraMachineConfigPoolCheck checks if a dedicated infrastructure machine config pool exists
type InfraMachineConfigPoolCheck struct {
	healthcheck.BaseCheck
}

// NewInfraMachineConfigPoolCheck creates a new infrastructure machine config pool check
func NewInfraMachineConfigPoolCheck() *InfraMachineConfigPoolCheck {
	return &InfraMachineConfigPoolCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infra-machine-config-pool",
			"Infrastructure Machine Config Pool",
			"Checks if a dedicated infrastructure machine config pool exists",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *InfraMachineConfigPoolCheck) Run() (healthcheck.Result, error) {
	// Check if infrastructure machine config pool exists
	out, err := utils.RunCommand("oc", "get", "machineconfigpool", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get machine config pools",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting machine config pools: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "machineconfigpool")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed machine config pool information"
	}

	// Check if infra pool exists
	hasInfraPool := strings.Contains(out, "infra")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	if !hasInfraPool {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No dedicated infrastructure machine config pool found",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Create a dedicated infrastructure machine config pool")
		result.AddRecommendation("In a production deployment, it is recommended that you deploy at least three machine sets to hold infrastructure components")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/machine_management/index#creating-infrastructure-machinesets", version))

		result.Detail = detailedOut
		return result, nil
	}

	// Check if the machine config pool is properly configured
	mcpStatus, err := utils.RunCommand("oc", "get", "mcp", "infra", "-o", "jsonpath={.status.conditions[?(@.type==\"Degraded\")].status}")
	if err != nil {
		// Not a critical error if we can't check the status
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Infrastructure machine config pool exists but status could not be determined",
			types.ResultKeyAdvisory,
		)
		result.Detail = detailedOut
		return result, nil
	}

	if strings.TrimSpace(mcpStatus) == "True" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Infrastructure machine config pool is degraded",
			types.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Dedicated infrastructure machine config pool is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
