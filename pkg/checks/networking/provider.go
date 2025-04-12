package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Get base networking checks from the renamed function in cni.go
	checks = append(checks, GetNetworkingChecks()...)

	// Add individual ingress controller checks
	checks = append(checks, NewIngressControllerTypeCheck())
	checks = append(checks, NewIngressControllerPlacementCheck())
	checks = append(checks, NewIngressControllerReplicaCheck())

	return checks
}
