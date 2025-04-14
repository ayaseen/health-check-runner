package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/utils"
	"strings"
)

// Helper function to check if OpenShift Logging is installed
func isLoggingInstalled() (bool, error) {
	// Check if the ClusterLogging CRD exists
	_, err := utils.RunCommand("oc", "get", "crd", "clusterloggings.logging.openshift.io")
	if err != nil {
		// The CRD doesn't exist, logging is not installed
		return false, nil
	}

	// Check if the namespace exists
	_, err = utils.RunCommand("oc", "get", "namespace", "openshift-logging")
	if err != nil {
		// The namespace doesn't exist, logging is not installed
		return false, nil
	}

	// Check if there's an instance deployed
	out, err := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging")
	if err != nil {
		// No ClusterLogging instance found
		return false, nil
	}

	// Check if "instance" is deployed
	return strings.Contains(out, "instance"), nil
}
