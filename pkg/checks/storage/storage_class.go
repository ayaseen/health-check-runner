/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for storage classes. It:

- Verifies if appropriate storage classes are configured
- Checks for a default storage class configuration
- Examines storage classes for ReadWriteMany (RWX) capability
- Provides recommendations for storage class configuration
- Helps ensure proper dynamic provisioning capabilities

This check helps maintain proper storage provisioning options for applications requiring persistent storage.
*/

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
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategoryStorage,
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
			types.StatusCritical,
			"Failed to get cluster config",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create a dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to create Kubernetes client",
			types.ResultKeyRequired,
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
			types.StatusCritical,
			"Failed to retrieve storage classes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving storage classes: %v", err)
	}

	// Get detailed information using oc command for the report
	detailedOut, err := utils.RunCommand("oc", "get", "storageclasses", "-o", "wide")

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Storage Classes Analysis ===\n\n")

	if err == nil && strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Storage Classes:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Storage Classes: No information available\n\n")
	}

	// Get detailed YAML output for more comprehensive information
	detailedYamlOut, err := utils.RunCommand("oc", "get", "storageclasses", "-o", "yaml")
	if err == nil && strings.TrimSpace(detailedYamlOut) != "" {
		formattedDetailOut.WriteString("Storage Classes (YAML):\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedYamlOut)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Check if any storage classes exist
	if len(storageClasses.Items) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No storage classes found",
			types.ResultKeyRecommended,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
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

	// Add storage class analysis section
	formattedDetailOut.WriteString("=== Storage Class Analysis ===\n\n")

	if defaultStorageClass != "" {
		formattedDetailOut.WriteString(fmt.Sprintf("Default Storage Class: %s\n\n", defaultStorageClass))
	} else {
		formattedDetailOut.WriteString("Default Storage Class: None\n\n")
	}

	formattedDetailOut.WriteString("Storage Class Names:\n")
	for _, name := range storageClassNames {
		formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", name))
	}
	formattedDetailOut.WriteString("\n")

	if hasRWXStorageClass {
		formattedDetailOut.WriteString("ReadWriteMany (RWX) Capable Storage: Available\n\n")
	} else {
		formattedDetailOut.WriteString("ReadWriteMany (RWX) Capable Storage: Not Detected\n\n")
	}

	// Create appropriate result based on findings
	if defaultStorageClass == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No default storage class is configured",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure a default storage class for dynamic provisioning")
		result.AddRecommendation("Use 'oc patch storageclass <name> -p '{\"metadata\":{\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'")

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// If no RWX storage class is available, add a warning
	if !hasRWXStorageClass {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Default storage class '%s' is configured, but no ReadWriteMany (RWX) capable storage class found", defaultStorageClass),
			types.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider adding a storage class that supports ReadWriteMany access mode for shared storage needs")
		result.Detail = formattedDetailOut.String()

		return result, nil
	}

	// All looks good
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Default storage class '%s' is configured and RWX-capable storage is available", defaultStorageClass),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
