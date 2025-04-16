/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for application probes. It:

- Identifies deployments and stateful sets lacking readiness and liveness probes
- Calculates the percentage of workloads with properly configured probes
- Provides detailed explanations about the importance of probes
- Recommends best practices for probe configuration
- Helps ensure application resilience and proper health monitoring

This check helps improve application availability and reliability by encouraging proper health check implementations.
*/

package applications

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProbesCheck checks if applications have readiness and liveness probes configured
type ProbesCheck struct {
	healthcheck.BaseCheck
}

// NewProbesCheck creates a new probes check
func NewProbesCheck() *ProbesCheck {
	return &ProbesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"application-probes",
			"Application Probes",
			"Checks if applications have readiness and liveness probes configured",
			types.CategoryApplications,
		),
	}
}

// Run executes the health check
func (c *ProbesCheck) Run() (healthcheck.Result, error) {
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

	// Counters for workloads with and without probes
	totalWorkloads := 0
	workloadsWithoutReadinessProbe := 0
	workloadsWithoutLivenessProbe := 0
	workloadsWithoutBothProbes := 0

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

	// Lists to collect details on workloads without probes
	var namespacesWithoutProbes []string
	var workloadsWithoutProbesDetails []string

	// Check each namespace for deployments
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") ||
			strings.HasPrefix(namespace.Name, "kube-") ||
			strings.Contains(namespace.Name, "operator") ||
			strings.Contains(namespace.Name, "multicluster") ||
			strings.Contains(namespace.Name, "open-cluster") {
			continue
		}

		// Get deployments in the namespace
		deployments, err := clientset.AppsV1().Deployments(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		// Check each deployment for probes
		namespaceMissingProbes := false

		for _, deployment := range deployments.Items {
			// Skip deployments with certain labels that might be system components or operators
			if isSystemDeployment(deployment) || strings.Contains(deployment.Name, "operator") {
				continue
			}

			totalWorkloads++

			// Check if the deployment has containers with readiness and liveness probes
			var missingReadinessProbe, missingLivenessProbe bool

			// Check each container in the deployment
			for _, container := range deployment.Spec.Template.Spec.Containers {
				if container.ReadinessProbe == nil {
					missingReadinessProbe = true
				}

				if container.LivenessProbe == nil {
					missingLivenessProbe = true
				}
			}

			// Update counters for missing probes
			if missingReadinessProbe {
				workloadsWithoutReadinessProbe++
			}

			if missingLivenessProbe {
				workloadsWithoutLivenessProbe++
			}

			if missingReadinessProbe && missingLivenessProbe {
				workloadsWithoutBothProbes++

				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- Deployment '%s' in namespace '%s' is missing both readiness and liveness probes",
						deployment.Name, namespace.Name))

				namespaceMissingProbes = true
			} else if missingReadinessProbe {
				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- Deployment '%s' in namespace '%s' is missing readiness probe",
						deployment.Name, namespace.Name))

				namespaceMissingProbes = true
			} else if missingLivenessProbe {
				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- Deployment '%s' in namespace '%s' is missing liveness probe",
						deployment.Name, namespace.Name))

				namespaceMissingProbes = true
			}
		}

		// Add namespace to the list if it has workloads without probes
		if namespaceMissingProbes {
			namespacesWithoutProbes = append(namespacesWithoutProbes, namespace.Name)
		}
	}

	// Check also StatefulSets
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") ||
			strings.HasPrefix(namespace.Name, "kube-") ||
			strings.Contains(namespace.Name, "operator") ||
			strings.Contains(namespace.Name, "multicluster") ||
			strings.Contains(namespace.Name, "open-cluster") {
			continue
		}

		// Get StatefulSets in the namespace
		statefulsets, err := clientset.AppsV1().StatefulSets(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		// Check each StatefulSet for probes
		namespaceMissingProbes := false

		for _, statefulSet := range statefulsets.Items {
			// Skip StatefulSets with certain labels that might be system components or operators
			if isSystemStatefulSet(statefulSet) || strings.Contains(statefulSet.Name, "operator") {
				continue
			}

			totalWorkloads++

			// Check if the StatefulSet has containers with readiness and liveness probes
			var missingReadinessProbe, missingLivenessProbe bool

			// Check each container in the StatefulSet
			for _, container := range statefulSet.Spec.Template.Spec.Containers {
				if container.ReadinessProbe == nil {
					missingReadinessProbe = true
				}

				if container.LivenessProbe == nil {
					missingLivenessProbe = true
				}
			}

			// Update counters for missing probes
			if missingReadinessProbe {
				workloadsWithoutReadinessProbe++
			}

			if missingLivenessProbe {
				workloadsWithoutLivenessProbe++
			}

			if missingReadinessProbe && missingLivenessProbe {
				workloadsWithoutBothProbes++

				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- StatefulSet '%s' in namespace '%s' is missing both readiness and liveness probes",
						statefulSet.Name, namespace.Name))

				namespaceMissingProbes = true
			} else if missingReadinessProbe {
				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- StatefulSet '%s' in namespace '%s' is missing readiness probe",
						statefulSet.Name, namespace.Name))

				namespaceMissingProbes = true
			} else if missingLivenessProbe {
				// Add details about the workload
				workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
					fmt.Sprintf("- StatefulSet '%s' in namespace '%s' is missing liveness probe",
						statefulSet.Name, namespace.Name))

				namespaceMissingProbes = true
			}
		}

		// Add namespace to the list if it has workloads without probes
		if namespaceMissingProbes && !contains(namespacesWithoutProbes, namespace.Name) {
			namespacesWithoutProbes = append(namespacesWithoutProbes, namespace.Name)
		}
	}

	// Check also DeploymentConfigs for OpenShift-specific workloads
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") ||
			strings.HasPrefix(namespace.Name, "kube-") ||
			strings.Contains(namespace.Name, "operator") ||
			strings.Contains(namespace.Name, "multicluster") ||
			strings.Contains(namespace.Name, "open-cluster") {
			continue
		}

		// Get DeploymentConfigs in the namespace (using oc command as client-go doesn't have direct access)
		dcOut, err := utils.RunCommand("oc", "get", "deploymentconfigs", "-n", namespace.Name, "-o", "json")
		if err != nil || !strings.Contains(dcOut, "items") {
			// No DeploymentConfigs or error, continue with other namespaces
			continue
		}

		// Parse the output and extract info about missing probes
		// For simplicity, we'll just check if any DeploymentConfigs exist and run a check
		if strings.Contains(dcOut, "\"kind\": \"DeploymentConfig\"") {
			// Use direct oc command to get the status of probes in DeploymentConfigs
			probeCheckCmd := fmt.Sprintf(`oc get dc -n %s -o jsonpath="{range .items[*]}{.metadata.name}{': readiness='}{range .spec.template.spec.containers[*]}{.readinessProbe}{', '}{end}{' liveness='}{range .spec.template.spec.containers[*]}{.livenessProbe}{', '}{end}{'\n'}{end}"`, namespace.Name)
			probeOut, err := utils.RunCommandWithInput("", "bash", "-c", probeCheckCmd)

			if err == nil {
				// Process the output to identify missing probes
				for _, line := range strings.Split(probeOut, "\n") {
					if line == "" {
						continue
					}

					parts := strings.Split(line, ":")
					if len(parts) < 2 {
						continue
					}

					dcName := parts[0]
					probeInfo := parts[1]

					// Skip operator-related deploymentconfigs
					if strings.Contains(dcName, "operator") {
						continue
					}

					totalWorkloads++

					// Check for missing probes
					missingReadiness := strings.Contains(probeInfo, "readiness=<nil>") || strings.Contains(probeInfo, "readiness=, ")
					missingLiveness := strings.Contains(probeInfo, "liveness=<nil>") || strings.Contains(probeInfo, "liveness=, ")

					if missingReadiness {
						workloadsWithoutReadinessProbe++
					}

					if missingLiveness {
						workloadsWithoutLivenessProbe++
					}

					if missingReadiness && missingLiveness {
						workloadsWithoutBothProbes++
						workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
							fmt.Sprintf("- DeploymentConfig '%s' in namespace '%s' is missing both readiness and liveness probes",
								dcName, namespace.Name))
						namespacesWithoutProbes = appendIfMissing(namespacesWithoutProbes, namespace.Name)
					} else if missingReadiness {
						workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
							fmt.Sprintf("- DeploymentConfig '%s' in namespace '%s' is missing readiness probe",
								dcName, namespace.Name))
						namespacesWithoutProbes = appendIfMissing(namespacesWithoutProbes, namespace.Name)
					} else if missingLiveness {
						workloadsWithoutProbesDetails = append(workloadsWithoutProbesDetails,
							fmt.Sprintf("- DeploymentConfig '%s' in namespace '%s' is missing liveness probe",
								dcName, namespace.Name))
						namespacesWithoutProbes = appendIfMissing(namespacesWithoutProbes, namespace.Name)
					}
				}
			}
		}
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Application Probes Analysis ===\n\n")

	// Add workload statistics with proper formatting
	formattedDetailOut.WriteString("Workload Statistics:\n")
	formattedDetailOut.WriteString(fmt.Sprintf("- Total User Workloads: %d\n", totalWorkloads))
	formattedDetailOut.WriteString(fmt.Sprintf("- Workloads Missing Readiness Probes: %d", workloadsWithoutReadinessProbe))

	if totalWorkloads > 0 {
		readinessProbePercentage := float64(workloadsWithoutReadinessProbe) / float64(totalWorkloads) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n", readinessProbePercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n")
	}

	formattedDetailOut.WriteString(fmt.Sprintf("- Workloads Missing Liveness Probes: %d", workloadsWithoutLivenessProbe))

	if totalWorkloads > 0 {
		livenessProbePercentage := float64(workloadsWithoutLivenessProbe) / float64(totalWorkloads) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n", livenessProbePercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n")
	}

	formattedDetailOut.WriteString(fmt.Sprintf("- Workloads Missing Both Probes: %d", workloadsWithoutBothProbes))

	if totalWorkloads > 0 {
		bothProbesPercentage := float64(workloadsWithoutBothProbes) / float64(totalWorkloads) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n\n", bothProbesPercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n\n")
	}

	// Add affected namespaces information with proper formatting
	if len(namespacesWithoutProbes) > 0 {
		formattedDetailOut.WriteString("Affected Namespaces:\n[source, text]\n----\n")
		for _, ns := range namespacesWithoutProbes {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", ns))
		}
		formattedDetailOut.WriteString("----\n\n")
	} else if totalWorkloads > 0 {
		formattedDetailOut.WriteString("Affected Namespaces: None (all namespaces have properly configured probes)\n\n")
	}

	// Add workload details with proper formatting
	if len(workloadsWithoutProbesDetails) > 0 {
		formattedDetailOut.WriteString("Workloads Missing Probes:\n[source, text]\n----\n")
		for _, detail := range workloadsWithoutProbesDetails {
			formattedDetailOut.WriteString(detail + "\n")
		}
		formattedDetailOut.WriteString("----\n\n")
	}

	// Add probe documentation
	formattedDetailOut.WriteString("=== Probe Information ===\n\n")
	formattedDetailOut.WriteString("What are Readiness and Liveness Probes?\n\n")
	formattedDetailOut.WriteString("Readiness Probe: Determines if a container is ready to accept traffic. When a pod's readiness check fails, it is removed from service load balancers.\n\n")
	formattedDetailOut.WriteString("Liveness Probe: Determines if a container is still running as expected. When a liveness check fails, Kubernetes will restart the container.\n\n")
	formattedDetailOut.WriteString("Benefits of using probes:\n")
	formattedDetailOut.WriteString("- Prevents traffic from being sent to unready containers\n")
	formattedDetailOut.WriteString("- Automatically restarts unhealthy containers\n")
	formattedDetailOut.WriteString("- Improves application resilience and availability\n")
	formattedDetailOut.WriteString("- Facilitates smoother deployments and updates\n")
	formattedDetailOut.WriteString("- Provides better visibility into application health\n\n")

	// If there are no workloads, return NotApplicable
	if totalWorkloads == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user workloads found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Calculate percentage of workloads without probes
	readinessProbePercentage := float64(workloadsWithoutReadinessProbe) / float64(totalWorkloads) * 100
	livenessProbePercentage := float64(workloadsWithoutLivenessProbe) / float64(totalWorkloads) * 100
	bothProbesPercentage := float64(workloadsWithoutBothProbes) / float64(totalWorkloads) * 100

	// If all workloads have both probes, the check passes
	if workloadsWithoutReadinessProbe == 0 && workloadsWithoutLivenessProbe == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("All %d user workloads have readiness and liveness probes configured", totalWorkloads),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with missing probes information
	var status types.Status
	var resultKey types.ResultKey
	var message string

	// Determine result status based on percentage of workloads without probes
	if bothProbesPercentage > 50 {
		// Critical if more than half of workloads are missing both probes
		status = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		message = fmt.Sprintf("%.1f%% of user workloads (%d out of %d) are missing both readiness and liveness probes",
			bothProbesPercentage, workloadsWithoutBothProbes, totalWorkloads)
	} else if readinessProbePercentage > 30 || livenessProbePercentage > 30 {
		// Warning if more than 30% of workloads are missing either probe
		status = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		message = fmt.Sprintf("Many user workloads are missing probes: %.1f%% missing readiness probes, %.1f%% missing liveness probes",
			readinessProbePercentage, livenessProbePercentage)
	} else {
		// Otherwise, just an advisory
		status = types.StatusWarning
		resultKey = types.ResultKeyAdvisory
		message = fmt.Sprintf("Some user workloads are missing probes: %d missing readiness probes, %d missing liveness probes",
			workloadsWithoutReadinessProbe, workloadsWithoutLivenessProbe)
	}

	result := healthcheck.NewResult(
		c.ID(),
		status,
		message,
		resultKey,
	)

	result.AddRecommendation("Configure readiness and liveness probes for all user workloads")
	result.AddRecommendation("Follow the Kubernetes documentation on pod lifecycle and probes: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes")

	result.Detail = formattedDetailOut.String()
	return result, nil
}

// appendIfMissing adds a string to a slice if it's not already present
func appendIfMissing(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
