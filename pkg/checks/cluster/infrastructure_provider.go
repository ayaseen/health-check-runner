package cluster

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategoryClusterConfig,
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
				types.StatusCritical,
				"Failed to get infrastructure provider",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting infrastructure provider: %v", err)
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	// Removed the unused variable 'version' that was retrieved here

	providerType := strings.TrimSpace(out)
	if providerType == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"No infrastructure provider type detected",
			types.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Infrastructure provider type: %s", providerType),
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
