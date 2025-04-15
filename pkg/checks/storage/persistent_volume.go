/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for persistent volumes. It:

- Examines the status of persistent volumes in the cluster
- Identifies volumes in failed, pending, or released states
- Provides recommendations for addressing problematic volumes
- Helps maintain a healthy storage environment
- Identifies potential storage issues before they impact applications

This check helps ensure proper storage availability and management in OpenShift clusters.
*/

package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentVolumeCheck checks persistent volume health
type PersistentVolumeCheck struct {
	healthcheck.BaseCheck
}

// NewPersistentVolumeCheck creates a new persistent volume check
func NewPersistentVolumeCheck() *PersistentVolumeCheck {
	return &PersistentVolumeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"persistent-volumes",
			"Persistent Volumes",
			"Checks the health of persistent volumes",
			types.CategoryStorage,
		),
	}
}

// Run executes the health check
func (c *PersistentVolumeCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get the list of persistent volumes
	ctx := context.Background()
	pvs, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve persistent volumes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving persistent volumes: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "pv", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed persistent volume information"
	}

	// Check PV status
	var failedPVs []string
	var pendingPVs []string
	var releasedPVs []string

	for _, pv := range pvs.Items {
		switch pv.Status.Phase {
		case "Failed":
			failedPVs = append(failedPVs, fmt.Sprintf("- %s (Reason: %s)", pv.Name, pv.Status.Reason))
		case "Pending":
			pendingPVs = append(pendingPVs, fmt.Sprintf("- %s", pv.Name))
		case "Released":
			releasedPVs = append(releasedPVs, fmt.Sprintf("- %s", pv.Name))
		}
	}

	// Create result based on PV statuses
	if len(failedPVs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Found %d failed persistent volumes", len(failedPVs)),
			types.ResultKeyRequired,
		)

		result.AddRecommendation("Investigate and fix the failed persistent volumes")
		result.AddRecommendation("Consider manually deleting and recreating the volumes if appropriate")

		detail := fmt.Sprintf("Failed persistent volumes:\n%s\n\n", strings.Join(failedPVs, "\n"))
		if len(pendingPVs) > 0 {
			detail += fmt.Sprintf("Pending persistent volumes:\n%s\n\n", strings.Join(pendingPVs, "\n"))
		}
		if len(releasedPVs) > 0 {
			detail += fmt.Sprintf("Released persistent volumes:\n%s\n\n", strings.Join(releasedPVs, "\n"))
		}
		detail += fmt.Sprintf("Detailed output:\n%s", detailedOut)

		result.Detail = detail

		return result, nil
	}

	if len(pendingPVs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Found %d pending persistent volumes", len(pendingPVs)),
			types.ResultKeyAdvisory,
		)

		result.AddRecommendation("Check why persistent volumes are in pending state")

		detail := fmt.Sprintf("Pending persistent volumes:\n%s\n\n", strings.Join(pendingPVs, "\n"))
		if len(releasedPVs) > 0 {
			detail += fmt.Sprintf("Released persistent volumes:\n%s\n\n", strings.Join(releasedPVs, "\n"))
		}
		detail += fmt.Sprintf("Detailed output:\n%s", detailedOut)

		result.Detail = detail

		return result, nil
	}

	if len(releasedPVs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Found %d released persistent volumes that could be reclaimed", len(releasedPVs)),
			types.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider reclaiming or deleting released volumes that are no longer needed")
		result.Detail = fmt.Sprintf("Released persistent volumes:\n%s\n\nDetailed output:\n%s",
			strings.Join(releasedPVs, "\n"), detailedOut)

		return result, nil
	}

	// If we get here, all PVs are in a good state
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %d persistent volumes are healthy", len(pvs.Items)),
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
