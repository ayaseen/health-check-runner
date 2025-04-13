package security

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// EtcdHealthCheck checks the health of the etcd cluster
type EtcdHealthCheck struct {
	healthcheck.BaseCheck
}

// NewEtcdHealthCheck creates a new etcd health check
func NewEtcdHealthCheck() *EtcdHealthCheck {
	return &EtcdHealthCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"etcd-health",
			"ETCD Health",
			"Checks the health of the etcd cluster",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *EtcdHealthCheck) Run() (healthcheck.Result, error) {
	// Check for degraded etcd operator status
	out, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "jsonpath={.status.conditions[?(@.type==\"Degraded\")].status}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get etcd operator status",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting etcd operator status: %v", err)
	}

	degraded := strings.TrimSpace(out)

	// Get detailed etcd operator information
	detailedOut, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed etcd operator information"
	}

	// Check for available etcd operator status
	availableOut, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "jsonpath={.status.conditions[?(@.type==\"Available\")].status}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get etcd operator availability status",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting etcd operator availability status: %v", err)
	}

	available := strings.TrimSpace(availableOut)

	// Check etcd cluster status by looking at specific metrics
	// For a simplified check, we'll just examine if the etcd service is running and if the pods are healthy
	etcdPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-etcd", "-l", "app=etcd")
	if err != nil {
		// Non-critical error, we can continue with the other information
		etcdPodsOut = "Failed to get etcd pods information"
	}

	// Check if all etcd pods are running
	allPodsRunning := true
	if !strings.Contains(etcdPodsOut, "Running") {
		allPodsRunning = false
	}

	// If etcd is degraded or not available, the check fails
	if degraded == "True" || available != "True" || !allPodsRunning {
		status := healthcheck.StatusCritical
		resultKey := healthcheck.ResultKeyRequired

		// Create detailed message based on the issues found
		var issues []string
		if degraded == "True" {
			issues = append(issues, "etcd operator is degraded")
		}
		if available != "True" {
			issues = append(issues, "etcd operator is not available")
		}
		if !allPodsRunning {
			issues = append(issues, "not all etcd pods are running")
		}

		// Create result with etcd issues
		result := healthcheck.NewResult(
			c.ID(),
			status,
			fmt.Sprintf("ETCD cluster has issues: %s", strings.Join(issues, ", ")),
			resultKey,
		)

		result.AddRecommendation("Investigate etcd issues using 'oc get co etcd -o yaml'")
		result.AddRecommendation("Check etcd pod logs using 'oc logs -n openshift-etcd etcd-<node-name>'")
		result.AddRecommendation("Consult the documentation at https://docs.openshift.com/container-platform/latest/scalability_and_performance/recommended-host-practices.html#recommended-etcd-practices_recommended-host-practices")

		// Add detailed information
		fullDetail := fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s", detailedOut, etcdPodsOut)
		result.Detail = fullDetail

		return result, nil
	}

	// If everything looks good, return OK
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"ETCD cluster is healthy",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s", detailedOut, etcdPodsOut)
	return result, nil
}
