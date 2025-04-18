/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements a health check for etcd backup configuration. It:

- Examines the cluster for scheduled backup jobs for etcd
- Identifies CronJobs that might be backing up etcd based on naming patterns
- Verifies the status of the etcd cluster operator
- Provides recommendations for proper etcd backup procedures
- Helps ensure data recovery capabilities are in place

This check is critical for disaster recovery preparedness, ensuring that etcd data can be restored in case of cluster failures.
*/

package security

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategorySecurity,
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
				types.StatusWarning,
				"No CronJobs found that might be backing up etcd",
				types.ResultKeyRecommended,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get CronJobs",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting CronJobs: %v", err)
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== ETCD Backup Analysis ===\n\n")

	// Add CronJobs information with proper formatting
	if strings.TrimSpace(cronJobsOut) != "" {
		formattedDetailOut.WriteString("CronJobs in the Cluster:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(cronJobsOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("CronJobs in the Cluster: No CronJobs found\n\n")
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

	// Add ETCD operator information with proper formatting
	if strings.TrimSpace(etcdCOOut) != "" {
		formattedDetailOut.WriteString("ETCD Cluster Operator Status:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(etcdCOOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("ETCD Cluster Operator Status: No information available\n\n")
	}

	// Add backup job status
	formattedDetailOut.WriteString("=== Backup Status ===\n\n")
	if len(etcdBackupJobs) > 0 {
		formattedDetailOut.WriteString("Possible ETCD Backup Jobs Found:\n")
		for _, job := range etcdBackupJobs {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", job))
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("No CronJobs found that might be backing up etcd\n\n")
		formattedDetailOut.WriteString("Regular etcd backups are critical for disaster recovery. Without backups, cluster recovery options are limited.\n\n")
	}

	// If backup jobs are found, the check passes
	if len(etcdBackupJobs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("Found %d CronJobs that might be backing up etcd", len(etcdBackupJobs)),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with recommendation to set up etcd backup
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		"No CronJobs found that might be backing up etcd",
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Set up regular etcd backups to protect against data loss")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/backing-up-etcd.html")

	result.Detail = formattedDetailOut.String()

	return result, nil
}
