package security

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// PrivilegedContainersCheck checks for workloads with elevated privileges
// This is a new name to avoid duplicate with existing ElevatedPrivilegesCheck
type PrivilegedContainersCheck struct {
	healthcheck.BaseCheck
}

// PrivilegedWorkload represents a workload with elevated privileges
type PrivilegedWorkload struct {
	Namespace    string
	ResourceType string
	ResourceName string
}

// NewPrivilegedContainersCheck creates a new privileged containers check
// Using a new name to avoid duplicate with existing NewElevatedPrivilegesCheck
func NewPrivilegedContainersCheck() *PrivilegedContainersCheck {
	return &PrivilegedContainersCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"privileged-containers",
			"Privileged Containers",
			"Checks for containers running with privileged security context",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *PrivilegedContainersCheck) Run() (healthcheck.Result, error) {
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

			// Check each container for privileged security context
			for _, container := range pod.Spec.Containers {
				if container.SecurityContext != nil &&
					container.SecurityContext.Privileged != nil &&
					*container.SecurityContext.Privileged {

					// Determine owner resource
					ownerType, ownerName := findOwnerResource(clientset, ctx, pod, namespace.Name)

					// Add to the list of privileged workloads
					privilegedWorkloads = append(privilegedWorkloads, PrivilegedWorkload{
						Namespace:    namespace.Name,
						ResourceType: ownerType,
						ResourceName: ownerName,
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
		detail := fmt.Sprintf("- %s in namespace '%s'", workload.ResourceName, workload.Namespace)

		switch workload.ResourceType {
		case "Deployment":
			deploymentDetails = append(deploymentDetails, detail)
		case "DeploymentConfig":
			dcDetails = append(dcDetails, detail)
		case "Pod":
			podDetails = append(podDetails, detail)
		default:
			otherDetails = append(otherDetails, fmt.Sprintf("- %s '%s' in namespace '%s'",
				workload.ResourceType, workload.ResourceName, workload.Namespace))
		}
	}

	// Combine all details
	var allDetails []string

	if len(deploymentDetails) > 0 {
		allDetails = append(allDetails, "Deployments with privileged containers:")
		allDetails = append(allDetails, deploymentDetails...)
		allDetails = append(allDetails, "")
	}

	if len(dcDetails) > 0 {
		allDetails = append(allDetails, "DeploymentConfigs with privileged containers:")
		allDetails = append(allDetails, dcDetails...)
		allDetails = append(allDetails, "")
	}

	if len(podDetails) > 0 {
		allDetails = append(allDetails, "Pods with privileged containers:")
		allDetails = append(allDetails, podDetails...)
		allDetails = append(allDetails, "")
	}

	if len(otherDetails) > 0 {
		allDetails = append(allDetails, "Other resources with privileged containers:")
		allDetails = append(allDetails, otherDetails...)
		allDetails = append(allDetails, "")
	}

	// Create result with privileged workloads information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Found %d workloads running with privileged containers", len(privilegedWorkloads)),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Review and remove privileged containers from user workloads unless absolutely necessary")
	result.AddRecommendation("Use restrictive SCCs for user workloads following the principle of least privilege")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#managing-pod-security-policies", version))

	result.Detail = strings.Join(allDetails, "\n")
	return result, nil
}

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

// findOwnerResource determines the top-level owner resource of a pod
func findOwnerResource(clientset *kubernetes.Clientset, ctx context.Context, pod metav1.Object, namespace string) (string, string) {
	// Check if the pod has owner references
	if len(pod.GetOwnerReferences()) == 0 {
		return "Pod", pod.GetName()
	}

	// Get the owner reference
	ownerRef := pod.GetOwnerReferences()[0]

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
