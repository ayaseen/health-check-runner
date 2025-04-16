/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for ServiceMonitor configurations. It:

- Verifies if ServiceMonitors are configured for application monitoring
- Checks if User Workload Monitoring is enabled
- Examines custom metrics collection setup
- Provides recommendations for proper monitoring configuration
- Helps ensure application metrics are being collected appropriately

This check helps administrators ensure proper monitoring of application workloads beyond the default system monitoring.
*/

package monitoring

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

// ServiceMonitorCheck checks if ServiceMonitors are configured for application monitoring
type ServiceMonitorCheck struct {
	healthcheck.BaseCheck
}

// NewServiceMonitorCheck creates a new service monitor check
func NewServiceMonitorCheck() *ServiceMonitorCheck {
	return &ServiceMonitorCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"service-monitors",
			"Service Monitors",
			"Checks if ServiceMonitors are configured for monitoring application metrics",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *ServiceMonitorCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes client config
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client configuration: %v", err)
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

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder

	// Add section for User Workload Monitoring
	formattedDetailOut.WriteString("=== User Workload Monitoring Status ===\n\n")

	// Check if user workload monitoring is enabled
	uwmEnabled, uwmStatus := checkUserWorkloadMonitoringEnabled()
	if uwmEnabled {
		formattedDetailOut.WriteString("User Workload Monitoring is ENABLED in the cluster\n\n")
	} else {
		formattedDetailOut.WriteString("User Workload Monitoring is NOT ENABLED in the cluster\n\n")
	}

	// Add detailed UWM status if available
	if strings.TrimSpace(uwmStatus) != "" {
		formattedDetailOut.WriteString("User Workload Monitoring Status:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(uwmStatus)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Define the ServiceMonitor resource
	groupVersion := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}

	// Check if ServiceMonitors exist in the cluster
	var serviceMonitors *unstructured.UnstructuredList
	serviceMonitors, err = client.Resource(groupVersion).Namespace("").List(context.TODO(), metav1.ListOptions{})

	// If the error is related to the CRD not being found, this means user workload monitoring isn't enabled
	if err != nil {
		// Add section for error information
		formattedDetailOut.WriteString("=== ServiceMonitor Check Results ===\n\n")
		formattedDetailOut.WriteString("Error: ServiceMonitor CRD not found. This typically means User Workload Monitoring is not properly enabled.\n\n")

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"User Workload Monitoring is not enabled in the cluster",
			types.ResultKeyRecommended,
		)

		// Get OpenShift version for documentation links
		version, verErr := utils.GetOpenShiftMajorMinorVersion()
		if verErr != nil {
			version = "4.10" // Default to a known version if we can't determine
		}

		result.AddRecommendation("Enable User Workload Monitoring to monitor application metrics")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Exclude ServiceMonitors in system namespaces
	excludedPrefixes := []string{"default", "openshift", "kube", "open"}
	var userServiceMonitors []unstructured.Unstructured

	for _, sm := range serviceMonitors.Items {
		namespace := sm.GetNamespace()
		isSystemNamespace := false

		for _, prefix := range excludedPrefixes {
			if strings.HasPrefix(namespace, prefix) {
				isSystemNamespace = true
				break
			}
		}

		if !isSystemNamespace {
			userServiceMonitors = append(userServiceMonitors, sm)
		}
	}

	// Generate detailed ServiceMonitor information
	formattedDetailOut.WriteString("=== ServiceMonitor Check Results ===\n\n")

	if len(userServiceMonitors) > 0 {
		// Get a list of ServiceMonitors with their namespaces
		formattedDetailOut.WriteString("User ServiceMonitors found in the cluster:\n\n")
		for _, sm := range userServiceMonitors {
			formattedDetailOut.WriteString(fmt.Sprintf("- Namespace: %s, Name: %s\n", sm.GetNamespace(), sm.GetName()))
		}

		// Get a sample ServiceMonitor YAML for reference if there's at least one
		if len(userServiceMonitors) > 0 {
			sampleSM, err := utils.RunCommand("oc", "get", "servicemonitor", userServiceMonitors[0].GetName(), "-n", userServiceMonitors[0].GetNamespace(), "-o", "yaml")
			if err == nil && strings.TrimSpace(sampleSM) != "" {
				formattedDetailOut.WriteString("\nSample ServiceMonitor configuration:\n[source, yaml]\n----\n")
				formattedDetailOut.WriteString(sampleSM)
				formattedDetailOut.WriteString("\n----\n\n")
			}
		}
	} else {
		formattedDetailOut.WriteString("No user ServiceMonitors found in the cluster\n\n")
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if there are any user ServiceMonitors
	if len(userServiceMonitors) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No ServiceMonitors found for application metrics monitoring",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Create ServiceMonitors for your applications to collect custom metrics")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#specifying-how-a-service-is-monitored", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// At least one user ServiceMonitor exists
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Found %d ServiceMonitors for application metrics monitoring", len(userServiceMonitors)),
		types.ResultKeyNoChange,
	)

	result.Detail = formattedDetailOut.String()
	return result, nil
}

// checkUserWorkloadMonitoringEnabled checks if the user workload monitoring is enabled
func checkUserWorkloadMonitoringEnabled() (bool, string) {
	// Check if the openshift-user-workload-monitoring namespace exists
	out, err := utils.RunCommand("oc", "get", "namespace", "openshift-user-workload-monitoring")
	if err != nil {
		// Try checking the cluster-monitoring-config ConfigMap
		cmOut, cmErr := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
		if cmErr != nil {
			return false, "Failed to get user workload monitoring status"
		}
		return strings.Contains(cmOut, "enableUserWorkload: true"), cmOut
	}

	// The namespace exists, check the configuration
	configOut, _ := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
	return true, fmt.Sprintf("Namespace Status:\n%s\n\nConfig Status:\n%s", out, configOut)
}
