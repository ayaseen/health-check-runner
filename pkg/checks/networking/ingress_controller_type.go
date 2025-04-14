package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerTypeCheck checks if the ingress controller is the default type
type IngressControllerTypeCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerTypeCheck creates a new ingress controller type check
func NewIngressControllerTypeCheck() *IngressControllerTypeCheck {
	return &IngressControllerTypeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-type",
			"Ingress Controller Type",
			"Checks if the ingress controller is of the recommended type",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerTypeCheck) Run() (healthcheck.Result, error) {
	out, err := utils.RunCommand("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", `jsonpath={.metadata.labels.ingresscontroller\.operator\.openshift\.io/owning-ingresscontroller}`)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get ingress controller type",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller type: %v", err)
	}

	ingressType := strings.TrimSpace(out)

	if ingressType == "default" {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Default OpenShift ingress controller is in use",
			types.ResultKeyNoChange,
		), nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("Non-default ingress controller type is in use: %s", ingressType),
		types.ResultKeyAdvisory,
	)

	result.AddRecommendation("Ensure the non-default ingress controller meets your requirements")

	return result, nil
}
