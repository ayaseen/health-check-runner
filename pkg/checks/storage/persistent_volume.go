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

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Persistent Volumes Analysis ===\n\n")

	if err == nil && strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Persistent Volumes Overview:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Persistent Volumes Overview: No information available\n\n")
	}

	// Get more detailed YAML output
	//detailedYamlOut, _ := utils.RunCommand("oc", "get", "pv", "-o", "yaml")
	//if strings.TrimSpace(detailedYamlOut) != "" {
	//	formattedDetailOut.WriteString("Persistent Volumes (YAML):\n[source, yaml]\n----\n")
	//	formattedDetailOut.WriteString(detailedYamlOut)
	//	formattedDetailOut.WriteString("\n----\n\n")
	//}

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

	// Add volume status analysis section
	formattedDetailOut.WriteString("=== Volume Status Analysis ===\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("Total Persistent Volumes: %d\n\n", len(pvs.Items)))

	if len(failedPVs) > 0 {
		formattedDetailOut.WriteString("Failed Persistent Volumes:\n")
		for _, pv := range failedPVs {
			formattedDetailOut.WriteString(pv + "\n")
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("Failed Persistent Volumes: None\n\n")
	}

	if len(pendingPVs) > 0 {
		formattedDetailOut.WriteString("Pending Persistent Volumes:\n")
		for _, pv := range pendingPVs {
			formattedDetailOut.WriteString(pv + "\n")
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("Pending Persistent Volumes: None\n\n")
	}

	if len(releasedPVs) > 0 {
		formattedDetailOut.WriteString("Released Persistent Volumes:\n")
		for _, pv := range releasedPVs {
			formattedDetailOut.WriteString(pv + "\n")
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("Released Persistent Volumes: None\n\n")
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

		result.Detail = formattedDetailOut.String()
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

		result.Detail = formattedDetailOut.String()
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
		result.Detail = formattedDetailOut.String()

		return result, nil
	}

	// If we get here, all PVs are in a good state
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %d persistent volumes are healthy", len(pvs.Items)),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
