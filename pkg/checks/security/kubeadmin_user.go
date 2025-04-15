/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements a health check for the presence of the kubeadmin user. It:

- Checks if the kubeadmin user still exists in the cluster
- Verifies if the kubeadmin secret exists in the kube-system namespace
- Provides recommendations for removing this temporary user
- Helps ensure proper security practices after initial installation
- Reduces the risk of unauthorized cluster access

This check enforces the security best practice of removing the default administrator account after proper identity providers are configured.
*/

package security

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// KubeadminUserCheck checks if the kubeadmin user still exists
type KubeadminUserCheck struct {
	healthcheck.BaseCheck
}

// NewKubeadminUserCheck creates a new kubeadmin user check
func NewKubeadminUserCheck() *KubeadminUserCheck {
	return &KubeadminUserCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"kubeadmin-user",
			"Kubeadmin User",
			"Checks if the kubeadmin user still exists",
			types.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *KubeadminUserCheck) Run() (healthcheck.Result, error) {
	// Check if the kubeadmin secret exists in kube-system namespace
	out, err := utils.RunCommand("oc", "get", "secrets", "kubeadmin", "-n", "kube-system")

	// If the command returns without error, the secret exists
	kubeadminExists := err == nil && strings.TrimSpace(out) != ""

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// If kubeadmin user still exists
	if kubeadminExists {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"The kubeadmin user still exists and should be removed for security reasons",
			types.ResultKeyRequired,
		)

		result.AddRecommendation("This user is for temporary post-installation steps and should be removed to avoid potential security breaches")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/removing-kubeadmin", version))

		result.Detail = out
		return result, nil
	}

	// Kubeadmin user has been removed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"The kubeadmin user has been removed",
		types.ResultKeyNoChange,
	)
	result.Detail = "Secret 'kubeadmin' not found in 'kube-system' namespace"
	return result, nil
}
