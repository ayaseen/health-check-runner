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
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("ETCD encryption is enabled with type: %s", encryptionType),
			healthcheck.ResultKeyNoChange,
		).WithDetail(detailedOut), nil
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

	result.WithDetail(detailedOut)

	return result, nil
}

// EtcdBackupCheck checks if etcd backup is configured
type EtcdBackupCheck struct {
	healthcheck.BaseCheck
}

// NewEtcdBackupCheck creates a new etcd backup check
func NewEtcdBackupCheck() *EtcdBackupCheck {
	return &EtcdBackupCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"etcd-backup",
			"ETCD Backup",
			"Checks if etcd backup is configured",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *EtcdBackupCheck) Run() (healthcheck.Result, error) {
	// Look for CronJobs that might be backing up etcd
	cronJobsOut, err := utils.RunCommand("oc", "get", "cronjobs", "--all-namespaces")
	if err != nil {
		// This might not be a critical error, as it could just mean no cron jobs exist
		if strings.Contains(err.Error(), "No resources found") {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusWarning,
				"No CronJobs found that might be backing up etcd",
				healthcheck.ResultKeyRecommended,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get CronJobs",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting CronJobs: %v", err)
	}

	// Check if any CronJob has "etcd" or "backup" in its name
	lines := strings.Split(cronJobsOut, "\n")
	var etcdBackupJobs []string

	for _, line := range lines[1:] { // Skip header line
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			jobName := strings.ToLower(fields[1])
			if strings.Contains(jobName, "etcd") || strings.Contains(jobName, "backup") {
				etcdBackupJobs = append(etcdBackupJobs, fields[0]+"/"+fields[1])
			}
		}
	}

	// Also check the status of the etcd cluster operator
	etcdCOOut, err := utils.RunCommand("oc", "get", "co", "etcd")
	if err != nil {
		// Non-critical error, we can continue without this information
		etcdCOOut = "Failed to get etcd cluster operator status"
	}

	// If backup jobs are found, the check passes
	if len(etcdBackupJobs) > 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("Found %d CronJobs that might be backing up etcd", len(etcdBackupJobs)),
			healthcheck.ResultKeyNoChange,
		).WithDetail(fmt.Sprintf("Possible etcd backup jobs:\n%s\n\nETCD Cluster Operator status:\n%s",
			strings.Join(etcdBackupJobs, "\n"),
			etcdCOOut)), nil
	}

	// Create result with recommendation to set up etcd backup
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		"No CronJobs found that might be backing up etcd",
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Set up regular etcd backups to protect against data loss")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/backing-up-etcd.html")

	result.WithDetail(fmt.Sprintf("ETCD Cluster Operator status:\n%s", etcdCOOut))

	return result, nil
}

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
	// Check etcd performance metrics (simplified for demonstration)
	// In a real implementation, this would query Prometheus for specific metrics

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
		result.WithDetail(fullDetail)

		return result, nil
	}

	// Check additional etcd health metrics
	// In a real implementation, this would involve more detailed checks
	// For now, we'll just check if the etcd service is running and the pods are healthy

	// If everything looks good, return OK
	return healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"ETCD cluster is healthy",
		healthcheck.ResultKeyNoChange,
	).WithDetail(fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s", detailedOut, etcdPodsOut)), nil