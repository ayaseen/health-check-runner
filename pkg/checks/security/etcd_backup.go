package security

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

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
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("Found %d CronJobs that might be backing up etcd", len(etcdBackupJobs)),
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = fmt.Sprintf("Possible etcd backup jobs:\n%s\n\nETCD Cluster Operator status:\n%s",
			strings.Join(etcdBackupJobs, "\n"),
			etcdCOOut)
		return result, nil
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

	result.Detail = fmt.Sprintf("ETCD Cluster Operator status:\n%s", etcdCOOut)

	return result, nil
}
