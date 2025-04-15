/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for ingress controller certificates. It:

- Verifies if a custom certificate is configured for the default ingress controller
- Checks if the certificate secret exists and is properly configured
- Provides recommendations for replacing the default self-signed certificate
- Helps avoid browser security warnings for applications
- Ensures proper TLS configuration for external traffic

This check helps maintain secure and trusted communication for applications exposed through the OpenShift router.
*/

package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// DefaultIngressCertificateCheck checks if a custom certificate is configured for the default ingress controller
type DefaultIngressCertificateCheck struct {
	healthcheck.BaseCheck
}

// NewDefaultIngressCertificateCheck creates a new default ingress certificate check
func NewDefaultIngressCertificateCheck() *DefaultIngressCertificateCheck {
	return &DefaultIngressCertificateCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"default-ingress-certificate",
			"Default Ingress Certificate",
			"Checks if a custom certificate is configured for the default ingress controller",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *DefaultIngressCertificateCheck) Run() (healthcheck.Result, error) {
	// Check if a custom certificate is configured
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.defaultCertificate}")

	customCertConfigured := err == nil && strings.TrimSpace(out) != "" && strings.TrimSpace(out) != "{}"

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Check if the certificate exists in the namespace
	var certExists bool
	if customCertConfigured {
		// Extract the secret name from the output if possible
		secretName := "unknown"
		secretNameOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.defaultCertificate.name}")
		if err == nil && strings.TrimSpace(secretNameOut) != "" {
			secretName = strings.TrimSpace(secretNameOut)
		}

		// Check if the secret exists
		secretOut, err := utils.RunCommand("oc", "get", "secret", secretName, "-n", "openshift-ingress")
		certExists = err == nil && strings.Contains(secretOut, secretName)
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Evaluate the certificate configuration
	if !customCertConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No custom certificate is configured for the default ingress controller",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure a custom certificate for the default ingress controller to avoid browser security warnings")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/security/certificates/replacing-default-ingress-certificate", version))
		result.Detail = detailedOut
		return result, nil
	}

	if !certExists {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Custom certificate is configured but the certificate secret may not exist",
			types.ResultKeyRequired,
		)
		result.AddRecommendation("Verify that the certificate secret exists in the openshift-ingress namespace")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/security/certificates/replacing-default-ingress-certificate", version))
		result.Detail = detailedOut
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Custom certificate is properly configured for the default ingress controller",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
