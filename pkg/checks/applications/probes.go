package applications

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
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
			healthcheck.CategoryApplications,
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
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve namespaces",
			healthcheck.ResultKeyRequired,
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
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") {
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
			// Skip deployments with certain labels that might be system components
			if isSystemDeployment(deployment) {
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
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") {
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
			// Skip StatefulSets with certain labels that might be system components
			if isSystemStatefulSet(statefulSet) {
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

	// If there are no workloads, return NotApplicable
	if totalWorkloads == 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusNotApplicable,
			"No user workloads found in the cluster",
			healthcheck.ResultKeyNotApplicable,
		), nil
	}

	// Calculate percentage of workloads without probes
	readinessProbePercentage := float64(workloadsWithoutReadinessProbe) / float64(totalWorkloads) * 100
	livenessProbePercentage := float64(workloadsWithoutLivenessProbe) / float64(totalWorkloads) * 100
	bothProbesPercentage := float64(workloadsWithoutBothProbes) / float64(totalWorkloads) * 100

	// Prepare a detailed description of what readiness and liveness probes are
	probeDescription := `
What are Readiness and Liveness Probes?

Readiness Probe: Determines if a container is ready to accept traffic. When a pod's readiness check fails, it is removed from service load balancers.

Liveness Probe: Determines if a container is still running as expected. When a liveness check fails, Kubernetes will restart the container.

Benefits of using probes:
- Prevents traffic from being sent to unready containers
- Automatically restarts unhealthy containers
- Improves application resilience and availability
- Facilitates smoother deployments and updates
- Provides better visibility into application health
`

	// If all workloads have both probes, the check passes
	if workloadsWithoutReadinessProbe == 0 && workloadsWithoutLivenessProbe == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("All %d user workloads have readiness and liveness probes configured", totalWorkloads),
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = probeDescription
		return result, nil
	}

	// Create result with missing probes information
	var status healthcheck.Status
	var resultKey healthcheck.ResultKey
	var message string

	// Determine result status based on percentage of workloads without probes
	if bothProbesPercentage > 50 {
		// Critical if more than half of workloads are missing both probes
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyRecommended
		message = fmt.Sprintf("%.1f%% of user workloads (%d out of %d) are missing both readiness and liveness probes",
			bothProbesPercentage, workloadsWithoutBothProbes, totalWorkloads)
	} else if readinessProbePercentage > 30 || livenessProbePercentage > 30 {
		// Warning if more than 30% of workloads are missing either probe
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyRecommended
		message = fmt.Sprintf("Many user workloads are missing probes: %.1f%% missing readiness probes, %.1f%% missing liveness probes",
			readinessProbePercentage, livenessProbePercentage)
	} else {
		// Otherwise, just an advisory
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyAdvisory
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

	// Add detailed information
	detail := fmt.Sprintf("Summary:\n"+
		"- Total user workloads: %d\n"+
		"- Workloads missing readiness probes: %d (%.1f%%)\n"+
		"- Workloads missing liveness probes: %d (%.1f%%)\n"+
		"- Workloads missing both probes: %d (%.1f%%)\n\n"+
		"Affected namespaces:\n- %s\n\n"+
		"Affected workloads:\n%s\n\n%s",
		totalWorkloads,
		workloadsWithoutReadinessProbe, readinessProbePercentage,
		workloadsWithoutLivenessProbe, livenessProbePercentage,
		workloadsWithoutBothProbes, bothProbesPercentage,
		strings.Join(namespacesWithoutProbes, "\n- "),
		strings.Join(workloadsWithoutProbesDetails, "\n"),
		probeDescription)

	result.Detail = detail

	return result, nil
}
