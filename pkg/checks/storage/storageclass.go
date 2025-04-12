package storage

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// StorageClassCheck checks if appropriate storage classes are configured
type StorageClassCheck struct {
	healthcheck.BaseCheck
}

// NewStorageClassCheck creates a new storage class check
func NewStorageClassCheck() *StorageClassCheck {
	return &StorageClassCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"storage-classes",
			"Storage Classes",
			"Checks if appropriate storage classes are configured",
			healthcheck.CategoryStorage,
		),
	}
}

// Run executes the health check
func (c *StorageClassCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes config
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get cluster config",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create a dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to create Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Define the StorageClass resource
	storageClassGVR := schema.GroupVersionResource{
		Group:    "storage.k8s.io",
		Version:  "v1",
		Resource: "storageclasses",
	}

	// Get the list of storage classes
	ctx := context.Background()
	storageClasses, err := client.Resource(storageClassGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve storage classes",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving storage classes: %v", err)
	}

	// Get detailed information using oc command for the report
	detailedOut, err := utils.RunCommand("oc", "get", "storageclasses", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed storage class information"
	}

	// Check if any storage classes exist
	if len(storageClasses.Items) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No storage classes found",
			healthcheck.ResultKeyRecommended,
		).WithDetail(detailedOut), nil
	}

	// Check if a default storage class is set
	var defaultStorageClass string
	var hasRWXStorageClass bool
	var storageClassNames []string

	for _, sc := range storageClasses.Items {
		name := sc.GetName()
		storageClassNames = append(storageClassNames, name)

		// Check if this is the default storage class
		annotations := sc.GetAnnotations()
		if annotations != nil {
			if val, ok := annotations["storageclass.kubernetes.io/is-default-class"]; ok && val == "true" {
				defaultStorageClass = name
			}
		}

		// Check if this storage class supports RWX access mode
		// This is a rough check and might need adjustments based on the actual storage provider
		unstructuredObj := sc.UnstructuredContent()
		provisioner, found, _ := unstructured.NestedString(unstructuredObj, "provisioner")
		if found {
			// Check known provisioners that support RWX
			if strings.Contains(provisioner, "nfs") ||
				strings.Contains(provisioner, "cephfs") ||
				strings.Contains(provisioner, "glusterfs") ||
				strings.Contains(provisioner, "azurefile") {
				hasRWXStorageClass = true
			}
		}
	}

	// Create appropriate result based on findings
	if defaultStorageClass == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No default storage class is configured",
			healthcheck.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure a default storage class for dynamic provisioning")
		result.AddRecommendation("Use 'oc patch storageclass <name> -p '{\"metadata\":{\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'")

		result.WithDetail(fmt.Sprintf("Available storage classes:\n%s\n\nDetailed output:\n%s",
			strings.Join(storageClassNames, ", "), detailedOut))

		return result, nil
	}

	// If no RWX storage class is available, add a warning
	if !hasRWXStorageClass {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Default storage class '%s' is configured, but no ReadWriteMany (RWX) capable storage class found", defaultStorageClass),
			healthcheck.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider adding a storage class that supports ReadWriteMany access mode for shared storage needs")
		result.WithDetail(fmt.Sprintf("Available storage classes:\n%s\n\nDetailed output:\n%s",
			strings.Join(storageClassNames, ", "), detailedOut))

		return result, nil
	}

	// All looks good
	return healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Default storage class '%s' is configured and RWX-capable storage is available", defaultStorageClass),
		healthcheck.ResultKeyNoChange,
	).WithDetail(detailedOut), nil
}

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
			healthcheck.CategoryStorage,
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
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get the list of persistent volumes
	ctx := context.Background()
	pvs, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve persistent volumes",
			healthcheck.ResultKeyRequired,
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
			healthcheck.StatusWarning,
			fmt.Sprintf("Found %d failed persistent volumes", len(failedPVs)),
			healthcheck.ResultKeyRequired,
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

		result.WithDetail(detail)

		return result, nil
	}

	if len(pendingPVs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Found %d pending persistent volumes", len(pendingPVs)),
			healthcheck.ResultKeyAdvisory,
		)

		result.AddRecommendation("Check why persistent volumes are in pending state")

		detail := fmt.Sprintf("Pending persistent volumes:\n%s\n\n", strings.Join(pendingPVs, "\n"))
		if len(releasedPVs) > 0 {
			detail += fmt.Sprintf("Released persistent volumes:\n%s\n\n", strings.Join(releasedPVs, "\n"))
		}
		detail += fmt.Sprintf("Detailed output:\n%s", detailedOut)

		result.WithDetail(detail)

		return result, nil
	}

	if len(releasedPVs) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Found %d released persistent volumes that could be reclaimed", len(releasedPVs)),
			healthcheck.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider reclaiming or deleting released volumes that are no longer needed")
		result.WithDetail(fmt.Sprintf("Released persistent volumes:\n%s\n\nDetailed output:\n%s",
			strings.Join(releasedPVs, "\n"), detailedOut))

		return result, nil
	}

	// If we get here, all PVs are in a good state
	return healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("All %d persistent volumes are healthy", len(pvs.Items)),
		healthcheck.ResultKeyNoChange,
	).WithDetail(detailedOut), nil
}

// StoragePerformanceCheck assesses storage performance characteristics
type StoragePerformanceCheck struct {
	healthcheck.BaseCheck
}

// NewStoragePerformanceCheck creates a new storage performance check
func NewStoragePerformanceCheck() *StoragePerformanceCheck {
	return &StoragePerformanceCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"storage-performance",
			"Storage Performance",
			"Assesses storage performance characteristics",
			healthcheck.CategoryStorage,
		),
	}
}

// Run executes the health check
func (c *StoragePerformanceCheck) Run() (healthcheck.Result, error) {
	// This is a placeholder for a more comprehensive storage performance check
	// In a production environment, this would likely include:
	// - Analysis of storage class QoS parameters
	// - Assessment of actual performance via metrics
	// - Evaluation of appropriate storage classes for different workloads

	// Get storage classes
	detailedOut, err := utils.RunCommand("oc", "get", "storageclasses", "-o", "yaml")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve storage classes",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving storage classes: %v", err)
	}

	// Check if we have any storage classes with performance annotations
	hasPerformanceClasses := strings.Contains(detailedOut, "performance") ||
		strings.Contains(detailedOut, "iops") ||
		strings.Contains(detailedOut, "throughput")

	if !hasPerformanceClasses {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No storage classes with explicit performance characteristics found",
			healthcheck.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider defining storage classes with different performance tiers")
		result.AddRecommendation("Label storage classes with performance characteristics for better workload placement")

		result.WithDetail(fmt.Sprintf("Storage Class Details:\n%s", detailedOut))

		return result, nil
	}

	return healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Storage classes with performance characteristics are available",
		healthcheck.ResultKeyNoChange,
	).WithDetail(detailedOut), nil
}
