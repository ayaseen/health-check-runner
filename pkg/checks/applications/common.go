/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file provides common utility functions used across application health checks. It includes:

- Functions to identify system pods and components versus user workloads
- Methods to determine if deployments and stateful sets are part of OpenShift system components
- Helper functions to examine labels and names to categorize resources
- Utilities for string operations used in application checks

These functions help separate OpenShift infrastructure components from user applications, ensuring that health checks focus on the appropriate resources.
*/

package applications

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
)

// isSystemPod checks if a pod is a system component based on its labels
func isSystemPod(labels map[string]string) bool {
	// Check if the pod has certain labels that indicate it's a system component
	if _, ok := labels["app.kubernetes.io/part-of"]; ok {
		if labels["app.kubernetes.io/part-of"] == "openshift" {
			return true
		}
	}

	// Check for other common system component labels
	if _, ok := labels["app"]; ok {
		if strings.Contains(labels["app"], "operator") ||
			strings.Contains(labels["app"], "controller") ||
			strings.Contains(labels["app"], "webhook") {
			return true
		}
	}

	return false
}

// isSystemDeployment checks if a deployment is a system component
func isSystemDeployment(deployment appsv1.Deployment) bool {
	// Check if the deployment has certain labels that indicate it's a system component
	if _, ok := deployment.Labels["app.kubernetes.io/part-of"]; ok {
		if deployment.Labels["app.kubernetes.io/part-of"] == "openshift" {
			return true
		}
	}

	// Check the name for common system component patterns
	if strings.Contains(deployment.Name, "operator") ||
		strings.Contains(deployment.Name, "controller") ||
		strings.Contains(deployment.Name, "webhook") {
		return true
	}

	return false
}

// isSystemStatefulSet checks if a StatefulSet is a system component
func isSystemStatefulSet(statefulSet appsv1.StatefulSet) bool {
	// Check if the StatefulSet has certain labels that indicate it's a system component
	if _, ok := statefulSet.Labels["app.kubernetes.io/part-of"]; ok {
		if statefulSet.Labels["app.kubernetes.io/part-of"] == "openshift" {
			return true
		}
	}

	// Check the name for common system component patterns
	if strings.Contains(statefulSet.Name, "operator") ||
		strings.Contains(statefulSet.Name, "controller") ||
		strings.Contains(statefulSet.Name, "webhook") {
		return true
	}

	return false
}

// contains checks if a string is in a slice
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
