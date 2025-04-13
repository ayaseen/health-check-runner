package security

import (
	"fmt"
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
			healthcheck.CategorySecurity,
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
			healthcheck.StatusCritical,
			"Failed to get etcd encryption configuration",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting etcd encryption configuration: %v", err)
	}

	encryptionType := strings.TrimSpace(out)

	// Get detailed information about the API server configuration
	detailedOut, err := utils.RunCommand("oc", "get", "apiserver", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed API server configuration"
	}

	// Check if encryption is enabled (aescbc or aesgcm)
	if encryptionType == "aescbc" || encryptionType == "aesgcm" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("ETCD encryption is enabled with type: %s", encryptionType),
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with recommendation to enable encryption
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		"ETCD encryption is not enabled",
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Enable etcd encryption to protect sensitive data")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/security/encrypting-etcd.html")

	result.Detail = detailedOut

	return result, nil
}
