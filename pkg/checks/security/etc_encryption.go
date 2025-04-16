/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements a health check for etcd encryption configuration. It:

- Verifies whether etcd encryption is enabled for sensitive data
- Checks the encryption type used (aescbc or aesgcm)
- Examines the API server configuration for encryption settings
- Provides recommendations for enabling encryption if not configured
- Helps ensure that sensitive data in etcd is properly protected

This check is important for data security, particularly for clusters storing sensitive information in their configuration resources.
*/

package security

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// EtcdEncryptionCheck checks if etcd encryption is enabled
type EtcdEncryptionCheck struct {
	healthcheck.BaseCheck
}

// NewEtcdEncryptionCheck creates a new etcd encryption check
func NewEtcdEncryptionCheck() *EtcdEncryptionCheck {
	return &EtcdEncryptionCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"etcd-encryption",
			"ETCD Encryption",
			"Checks if etcd encryption is enabled for sensitive data",
			types.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *EtcdEncryptionCheck) Run() (healthcheck.Result, error) {
	// Get the encryption type of the etcd server
	out, err := utils.RunCommand("oc", "get", "apiserver", "-o", "jsonpath={.items[*].spec.encryption.type}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get etcd encryption configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting etcd encryption configuration: %v", err)
	}

	encryptionType := strings.TrimSpace(out)

	// Get detailed information about the API server configuration
	detailedOut, err := utils.RunCommand("oc", "get", "apiserver", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed API server configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== ETCD Encryption Analysis ===\n\n")

	// Add API server configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("API Server Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("API Server Configuration: No information available\n\n")
	}

	// Add encryption status
	formattedDetailOut.WriteString("=== Encryption Status ===\n\n")
	if encryptionType == "aescbc" || encryptionType == "aesgcm" {
		formattedDetailOut.WriteString(fmt.Sprintf("ETCD encryption is enabled with type: %s\n\n", encryptionType))
		formattedDetailOut.WriteString("This ensures that sensitive data stored in etcd is properly encrypted.\n\n")
	} else {
		formattedDetailOut.WriteString("ETCD encryption is NOT enabled\n\n")
		formattedDetailOut.WriteString("Without encryption, sensitive data in etcd is stored in plaintext, which could be a security risk.\n\n")
	}

	// Check if encryption is enabled (aescbc or aesgcm)
	if encryptionType == "aescbc" || encryptionType == "aesgcm" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("ETCD encryption is enabled with type: %s", encryptionType),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with recommendation to enable encryption
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		"ETCD encryption is not enabled",
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Enable etcd encryption to protect sensitive data")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/security/encrypting-etcd.html")

	result.Detail = formattedDetailOut.String()

	return result, nil
}
