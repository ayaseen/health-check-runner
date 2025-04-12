package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// InfrastructureProviderCheck checks the infrastructure provider configuration
type InfrastructureProviderCheck struct {
	healthcheck.BaseCheck
}

// NewInfrastructureProviderCheck creates a new infrastructure provider check
func NewInfrastructureProviderCheck() *InfrastructureProviderCheck {
	return &InfrastructureProviderCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infrastructure-provider",
			"Infrastructure Provider",
			"Checks the infrastructure provider configuration",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *InfrastructureProviderCheck) Run() (healthcheck.Result, error) {
	// Get the infrastructure provider type
	out, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.spec.platformSpec.type}")
	if err != nil {
		// Try alternative path in newer versions
		out, err = utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platform}")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusCritical,
				"Failed to get infrastructure provider",
				healthcheck.ResultKeyRequired,
			), fmt.Errorf("error getting infrastructure provider: %v", err)
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	providerType := strings.TrimSpace(out)
	if providerType == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No infrastructure provider type detected",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Infrastructure provider type: %s", providerType),
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
