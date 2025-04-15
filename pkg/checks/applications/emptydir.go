/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for applications using emptyDir volumes. It:

- Identifies deployments, stateful sets, and pods using emptyDir volumes
- Calculates the percentage of workloads using non-persistent storage
- Provides detailed explanations about the risks of using emptyDir
- Recommends alternatives for persistent storage needs
- Flags potential data loss scenarios due to pod rescheduling

This check helps administrators identify applications at risk of data loss due to the ephemeral nature of emptyDir volumes.
*/

package applications

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// EmptyDirVolumeCheck checks for applications using emptyDir volumes
type EmptyDirVolumeCheck struct {
	healthcheck.BaseCheck
}

// NewEmptyDirVolumeCheck creates a new empty directory volume check
func NewEmptyDirVolumeCheck() *EmptyDirVolumeCheck {
	return &EmptyDirVolumeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"emptydir-volumes",
			"EmptyDir Volumes",
			"Checks for applications using emptyDir volumes, which are ephemeral and not recommended for persistent data",
			types.CategoryApplications,
		),
	}
}

// Run executes the health check
func (c *EmptyDirVolumeCheck) Run() (healthcheck.Result, error) {
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

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve namespaces",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving namespaces: %v", err)
	}

	// Counters for workloads using emptyDir volumes
	totalWorkloads := 0
	workloadsWithEmptyDir := 0

	// Namespaces to skip (system namespaces)
	skipNamespaces := map[string]bool{
		"default":             true,
		"kube-system":         true,
		"kube-public":         true,
		"kube-node-lease":     true,
		"openshift":           true,
		"openshift-etcd":      true,
		"openshift-apiserver": true,
	}

	// Lists to collect details
	var workloadsWithEmptyDirDetails []string
	var namespacesWithEmptyDir []string

	// Check each namespace for pods with emptyDir volumes
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") {
			continue
		}

		// Get pods in the namespace
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		// Flag to track if this namespace has any pods with emptyDir
		namespaceHasEmptyDir := false

		for _, pod := range pods.Items {
			// Skip pods with certain labels that might be system components
			if isSystemPod(pod.Labels) {
				continue
			}

			totalWorkloads++

			// Check each volume in the pod
			hasEmptyDir := false
			for _, volume := range pod.Spec.Volumes {
				if volume.EmptyDir != nil {
					hasEmptyDir = true
					break
				}
			}

			if hasEmptyDir {
				workloadsWithEmptyDir++

				// Add details about the workload
				workloadsWithEmptyDirDetails = append(workloadsWithEmptyDirDetails,
					fmt.Sprintf("- Pod '%s' in namespace '%s' is using emptyDir volume",
						pod.Name, namespace.Name))

				namespaceHasEmptyDir = true
			}
		}

		// Get deployments in the namespace to check their templates
		deployments, err := clientset.AppsV1().Deployments(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		for _, deployment := range deployments.Items {
			// Skip deployments with certain labels that might be system components
			if isSystemDeployment(deployment) {
				continue
			}

			// Check volumes in the pod template
			hasEmptyDir := false
			for _, volume := range deployment.Spec.Template.Spec.Volumes {
				if volume.EmptyDir != nil {
					hasEmptyDir = true
					break
				}
			}

			if hasEmptyDir {
				workloadsWithEmptyDir++

				// Add details about the workload
				workloadsWithEmptyDirDetails = append(workloadsWithEmptyDirDetails,
					fmt.Sprintf("- Deployment '%s' in namespace '%s' is using emptyDir volume",
						deployment.Name, namespace.Name))

				namespaceHasEmptyDir = true
			}
		}

		// Get StatefulSets in the namespace to check their templates
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		for _, statefulset := range statefulsets.Items {
			// Skip StatefulSets with certain labels that might be system components
			if isSystemStatefulSet(statefulset) {
				continue
			}

			// Check volumes in the pod template
			hasEmptyDir := false
			for _, volume := range statefulset.Spec.Template.Spec.Volumes {
				if volume.EmptyDir != nil {
					hasEmptyDir = true
					break
				}
			}

			if hasEmptyDir {
				workloadsWithEmptyDir++

				// Add details about the workload
				workloadsWithEmptyDirDetails = append(workloadsWithEmptyDirDetails,
					fmt.Sprintf("- StatefulSet '%s' in namespace '%s' is using emptyDir volume",
						statefulset.Name, namespace.Name))

				namespaceHasEmptyDir = true
			}
		}

		// Add namespace to the list if it has workloads with emptyDir volumes
		if namespaceHasEmptyDir {
			namespacesWithEmptyDir = append(namespacesWithEmptyDir, namespace.Name)
		}
	}

	// If there are no workloads, return NotApplicable
	if totalWorkloads == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user workloads found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// If no workloads are using emptyDir volumes, the check passes
	if workloadsWithEmptyDir == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"No user workloads are using emptyDir volumes",
			types.ResultKeyNoChange,
		), nil
	}

	// Calculate percentage of workloads using emptyDir volumes
	emptyDirPercentage := float64(workloadsWithEmptyDir) / float64(totalWorkloads) * 100

	// Prepare a detailed description of emptyDir volumes
	emptyDirDescription := `
What are emptyDir Volumes?

An emptyDir volume is created when a Pod is assigned to a node, and exists as long as that Pod is running on that node. When a Pod is removed from a node for any reason, the data in the emptyDir is deleted permanently.

Risks of using emptyDir volumes:
- Data loss: All data is lost when the pod is deleted or rescheduled
- No persistence across pod restarts or rescheduling
- Not suitable for stateful applications that need data persistence
- No data sharing between different pods or nodes

Recommended alternatives:
- PersistentVolumeClaims (PVCs) for persistent storage
- ConfigMaps or Secrets for configuration data
- External storage services for important data
`

	// Create result based on the percentage of workloads using emptyDir volumes
	var status types.Status
	var resultKey types.ResultKey
	var message string

	// Determine result status based on percentage of workloads using emptyDir volumes
	if emptyDirPercentage > 50 {
		// Warning if more than half of workloads use emptyDir volumes
		status = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		message = fmt.Sprintf("%.1f%% of user workloads (%d out of %d) are using emptyDir volumes",
			emptyDirPercentage, workloadsWithEmptyDir, totalWorkloads)
	} else {
		// Otherwise, just an advisory
		status = types.StatusWarning
		resultKey = types.ResultKeyAdvisory
		message = fmt.Sprintf("%d user workloads are using emptyDir volumes", workloadsWithEmptyDir)
	}

	result := healthcheck.NewResult(
		c.ID(),
		status,
		message,
		resultKey,
	)

	result.AddRecommendation("Use persistent volumes instead of emptyDir for data that needs to persist")
	result.AddRecommendation("Review existing workloads using emptyDir to ensure they don't store important data")
	result.AddRecommendation("Follow the Kubernetes documentation on volumes: https://kubernetes.io/docs/concepts/storage/volumes/")

	// Add detailed information
	detail := fmt.Sprintf("Summary:\n"+
		"- Total user workloads: %d\n"+
		"- Workloads using emptyDir volumes: %d (%.1f%%)\n\n"+
		"Affected namespaces:\n- %s\n\n"+
		"Affected workloads:\n%s\n\n%s",
		totalWorkloads,
		workloadsWithEmptyDir, emptyDirPercentage,
		strings.Join(namespacesWithEmptyDir, "\n- "),
		strings.Join(workloadsWithEmptyDirDetails, "\n"),
		emptyDirDescription)

	result.Detail = detail

	return result, nil
}
