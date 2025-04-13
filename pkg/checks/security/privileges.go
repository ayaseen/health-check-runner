package security

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ElevatedPrivilegesCheck checks for workloads with elevated privileges
type ElevatedPrivilegesCheck struct {
	healthcheck.BaseCheck
}

// PrivilegedWorkload represents a workload with elevated privileges
type PrivilegedWorkload struct {
	Namespace    string
	ResourceType string
	ResourceName string
	Reason       string
}

// NewElevatedPrivilegesCheck creates a new elevated privileges check
func NewElevatedPrivilegesCheck() *ElevatedPrivilegesCheck {
	return &ElevatedPrivilegesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"elevated-privileges",
			"Elevated Privileges",
			"Checks for workloads running with elevated privileges",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *ElevatedPrivilegesCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get cluster config",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to create Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to list namespaces",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error listing namespaces: %v", err)
	}

	var privilegedWorkloads []PrivilegedWorkload

	// Check all namespaces
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if isSystemNamespace(namespace.Name) {
			continue
		}

		// Check pods in the namespace
		pods, err := clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		// Check each pod for privileged containers
		for _, pod := range pods.Items {
			// Skip build or deploy pods
			if isBuildOrDeployPod(pod.Name) {
				continue
			}

			// Check for privileged security context
			for _, container := range pod.Spec.Containers {
				if hasElevatedPrivileges(container) {
					// Determine owner resource
					ownerType, ownerName := findOwnerResource(clientset, ctx, &pod, namespace.Name)

					reason := "Privileged container"
					if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
						reason = "Privileged flag set to true"
					} else if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil && len(container.SecurityContext.Capabilities.Add) > 0 {
						reason = fmt.Sprintf("Added capabilities: %v", container.SecurityContext.Capabilities.Add)
					} else if container.SecurityContext != nil && container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
						reason = "Running as root (uid=0)"
					}

					// Add to the list of privileged workloads
					privilegedWorkloads = append(privilegedWorkloads, PrivilegedWorkload{
						Namespace:    namespace.Name,
						ResourceType: ownerType,
						ResourceName: ownerName,
						Reason:       reason,
					})

					// We found a privileged container in this pod, no need to check others
					break
				}
			}
		}
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// If no privileged workloads found
	if len(privilegedWorkloads) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"No user workloads using privileged containers were found",
			healthcheck.ResultKeyNoChange,
		)
		return result, nil
	}

	// Create detail strings for different workload types
	var deploymentDetails []string
	var dcDetails []string
	var podDetails []string
	var otherDetails []string

	for _, workload := range privilegedWorkloads {
		detail := fmt.Sprintf("- %s in namespace '%s' (%s)", workload.ResourceName, workload.Namespace, workload.Reason)

		switch workload.ResourceType {
		case "Deployment":
			deploymentDetails = append(deploymentDetails, detail)
		case "DeploymentConfig":
			dcDetails = append(dcDetails, detail)
		case "Pod":
			podDetails = append(podDetails, detail)
		default:
			otherDetails = append(otherDetails, fmt.Sprintf("- %s '%s' in namespace '%s' (%s)",
				workload.ResourceType, workload.ResourceName, workload.Namespace, workload.Reason))
		}
	}

	// Combine all details
	var allDetails []string

	if len(deploymentDetails) > 0 {
		allDetails = append(allDetails, "Deployments with elevated privileges:")
		allDetails = append(allDetails, deploymentDetails...)
		allDetails = append(allDetails, "")
	}

	if len(dcDetails) > 0 {
		allDetails = append(allDetails, "DeploymentConfigs with elevated privileges:")
		allDetails = append(allDetails, dcDetails...)
		allDetails = append(allDetails, "")
	}

	if len(podDetails) > 0 {
		allDetails = append(allDetails, "Pods with elevated privileges:")
		allDetails = append(allDetails, podDetails...)
		allDetails = append(allDetails, "")
	}

	if len(otherDetails) > 0 {
		allDetails = append(allDetails, "Other resources with elevated privileges:")
		allDetails = append(allDetails, otherDetails...)
		allDetails = append(allDetails, "")
	}

	// Create result with privileged workloads information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Found %d workloads running with elevated privileges", len(privilegedWorkloads)),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Review and remove privileged containers from user workloads unless absolutely necessary")
	result.AddRecommendation("Use restrictive SCCs for user workloads following the principle of least privilege")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#managing-pod-security-policies", version))

	result.Detail = strings.Join(allDetails, "\n")
	return result, nil
}

// Helper functions

// isSystemNamespace determines if a namespace should be excluded from checks
func isSystemNamespace(namespace string) bool {
	excludedPrefixes := []string{"openshift", "default", "kube", "open"}

	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(namespace, prefix) {
			return true
		}
	}

	return false
}

// isBuildOrDeployPod determines if a pod should be excluded from checks
func isBuildOrDeployPod(podName string) bool {
	excludedSuffixes := []string{"-build", "-deploy"}

	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(podName, suffix) {
			return true
		}
	}

	return false
}

// hasElevatedPrivileges checks if a container has elevated privileges
func hasElevatedPrivileges(container corev1.Container) bool {
	// Check for privileged flag
	if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
		return true
	}

	// Check for capabilities
	if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
		for _, cap := range container.SecurityContext.Capabilities.Add {
			// Check for dangerous capabilities
			if cap == "SYS_ADMIN" || cap == "NET_ADMIN" || cap == "ALL" {
				return true
			}
		}
	}

	// Check for running as root
	if container.SecurityContext != nil && container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
		return true
	}

	return false
}

// findOwnerResource determines the top-level owner resource of a pod
func findOwnerResource(clientset *kubernetes.Clientset, ctx context.Context, pod *corev1.Pod, namespace string) (string, string) {
	// Check if the pod has owner references
	if len(pod.OwnerReferences) == 0 {
		return "Pod", pod.Name
	}

	// Get the owner reference
	ownerRef := pod.OwnerReferences[0]

	switch ownerRef.Kind {
	case "ReplicaSet":
		// Check if the ReplicaSet is owned by a Deployment
		rs, err := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
		if err != nil || len(rs.OwnerReferences) == 0 {
			return "ReplicaSet", ownerRef.Name
		}

		deployOwnerRef := rs.OwnerReferences[0]
		if deployOwnerRef.Kind == "Deployment" {
			return "Deployment", deployOwnerRef.Name
		}
		return "ReplicaSet", ownerRef.Name

	case "ReplicationController":
		// In OpenShift, ReplicationControllers might be created by DeploymentConfigs
		// Try to determine if this RC belongs to a DeploymentConfig by checking its labels
		rc, err := clientset.CoreV1().ReplicationControllers(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
		if err != nil {
			return "ReplicationController", ownerRef.Name
		}

		// Check for DeploymentConfig labels
		if dcName, ok := rc.Labels["deploymentconfig"]; ok {
			return "DeploymentConfig", dcName
		}
		return "ReplicationController", ownerRef.Name

	case "StatefulSet":
		return "StatefulSet", ownerRef.Name

	case "DaemonSet":
		return "DaemonSet", ownerRef.Name

	case "Job":
		// Check if the Job is owned by a CronJob
		job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
		if err != nil || len(job.OwnerReferences) == 0 {
			return "Job", ownerRef.Name
		}

		cronJobOwnerRef := job.OwnerReferences[0]
		if cronJobOwnerRef.Kind == "CronJob" {
			return "CronJob", cronJobOwnerRef.Name
		}
		return "Job", ownerRef.Name

	default:
		return ownerRef.Kind, ownerRef.Name
	}
}
