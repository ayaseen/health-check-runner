/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for user workload monitoring. It:

- Verifies if monitoring for user-defined projects is enabled
- Checks the cluster-monitoring-config ConfigMap for proper settings
- Examines the existence of the openshift-user-workload-monitoring namespace
- Verifies correct configuration of required monitoring components for user workloads
- Provides recommendations for enabling application monitoring
- Helps ensure proper visibility into application performance

This check helps administrators configure proper monitoring for application workloads beyond the default system monitoring.
*/

package monitoring

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// UserWorkloadMonitoringCheck checks if user-defined workload monitoring is enabled
type UserWorkloadMonitoringCheck struct {
	healthcheck.BaseCheck
}

// NewUserWorkloadMonitoringCheck creates a new user workload monitoring check
func NewUserWorkloadMonitoringCheck() *UserWorkloadMonitoringCheck {
	return &UserWorkloadMonitoringCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"user-workload-monitoring",
			"User Workload Monitoring",
			"Checks if monitoring for user-defined projects is enabled and properly configured",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *UserWorkloadMonitoringCheck) Run() (healthcheck.Result, error) {
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

	// Create dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to create Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Check if the openshift-monitoring namespace exists
	monitoringNamespace, err := checkNamespaceExists(client, "openshift-monitoring")
	if err != nil || !monitoringNamespace {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get openshift-monitoring namespace",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting openshift-monitoring namespace: %v", err)
	}

	// Check if the cluster-monitoring-config ConfigMap exists
	monConfigExists, monConfigYaml := getClusterMonitoringConfig(client)

	// Check if enableUserWorkload is set to true in the cluster-monitoring-config ConfigMap
	uwmEnabled := false
	if monConfigExists {
		uwmEnabled = strings.Contains(monConfigYaml, "enableUserWorkload: true")
	}

	// Check if the openshift-user-workload-monitoring namespace exists
	uwmNamespaceExists, err := checkNamespaceExists(client, "openshift-user-workload-monitoring")
	if err != nil {
		// This might not be a critical error, so we'll just note it and continue
		uwmNamespaceExists = false
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.14" // Default to a known version if we can't determine
	}

	// Create detailed information for the report
	var detailedOut strings.Builder
	detailedOut.WriteString("== User Workload Monitoring Configuration ==\n\n")

	if monConfigExists {
		detailedOut.WriteString("Cluster Monitoring Config exists:\n\n")
		detailedOut.WriteString("[source, yaml]\n----\n")
		detailedOut.WriteString(monConfigYaml)
		detailedOut.WriteString("\n----\n\n")
		detailedOut.WriteString(fmt.Sprintf("User Workload Monitoring enabled in config: %v\n", uwmEnabled))
	} else {
		detailedOut.WriteString("No cluster-monitoring-config ConfigMap found\n\n")
	}

	detailedOut.WriteString(fmt.Sprintf("User Workload Monitoring namespace exists: %v\n\n", uwmNamespaceExists))

	// If user workload monitoring is enabled, check the components
	if uwmEnabled && uwmNamespaceExists {
		// Get the user-workload-monitoring-config ConfigMap
		uwmConfigExists, uwmConfigYaml := getUserWorkloadMonitoringConfig(client)

		detailedOut.WriteString("== User Workload Monitoring Components ==\n\n")

		if uwmConfigExists {
			detailedOut.WriteString("User Workload Monitoring Config exists:\n\n")
			detailedOut.WriteString("[source, yaml]\n----\n")
			detailedOut.WriteString(uwmConfigYaml)
			detailedOut.WriteString("\n----\n\n")

			// Check user workload monitoring components
			configuredComponents, missingComponents, componentDetails := checkUserWorkloadComponents(uwmConfigYaml)
			detailedOut.WriteString(componentDetails)

			if len(missingComponents) > 0 {
				result := healthcheck.NewResult(
					c.ID(),
					types.StatusWarning,
					fmt.Sprintf("User Workload Monitoring is enabled but missing configuration for components: %s", strings.Join(missingComponents, ", ")),
					types.ResultKeyRecommended,
				)

				result.AddRecommendation("Configure all required components for User Workload Monitoring")
				result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#configuring-the-monitoring-stack", version))

				result.Detail = detailedOut.String()
				return result, nil
			}

			// Check for persistent storage configuration
			hasPersistentStorage, storageDetails := checkUserWorkloadPersistentStorage(uwmConfigYaml)
			detailedOut.WriteString("\n== User Workload Monitoring Storage ==\n\n")
			detailedOut.WriteString(storageDetails)

			if !hasPersistentStorage {
				result := healthcheck.NewResult(
					c.ID(),
					types.StatusWarning,
					"User Workload Monitoring is enabled but persistent storage is not configured",
					types.ResultKeyAdvisory,
				)

				result.AddRecommendation("Configure persistent storage for User Workload Monitoring components")
				result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#configuring-persistent-storage", version))

				result.Detail = detailedOut.String()
				return result, nil
			}

			// All checks passed
			result := healthcheck.NewResult(
				c.ID(),
				types.StatusOK,
				fmt.Sprintf("User Workload Monitoring is enabled and properly configured with %d components", len(configuredComponents)),
				types.ResultKeyNoChange,
			)

			result.Detail = detailedOut.String()
			return result, nil
		} else {
			// User workload monitoring is enabled but config map not found
			detailedOut.WriteString("No user-workload-monitoring-config ConfigMap found, using default configuration\n\n")

			// Check if components are running
			running, componentStatus := checkUserWorkloadComponentsRunning()
			detailedOut.WriteString(componentStatus)

			if running {
				result := healthcheck.NewResult(
					c.ID(),
					types.StatusOK,
					"User Workload Monitoring is enabled with default configuration",
					types.ResultKeyNoChange,
				)

				result.Detail = detailedOut.String()
				return result, nil
			} else {
				result := healthcheck.NewResult(
					c.ID(),
					types.StatusWarning,
					"User Workload Monitoring is enabled but some components are not running",
					types.ResultKeyRecommended,
				)

				result.AddRecommendation("Check User Workload Monitoring component status")
				result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#investigating-why-user-defined-metrics-are-unavailable", version))

				result.Detail = detailedOut.String()
				return result, nil
			}
		}
	}

	// User workload monitoring is not enabled
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		"User Workload Monitoring is not enabled",
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Enable monitoring for user-defined projects")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects", version))

	result.Detail = detailedOut.String()
	return result, nil
}

// checkNamespaceExists checks if a namespace exists
func checkNamespaceExists(client dynamic.Interface, namespace string) (bool, error) {
	// Define the Namespace resource
	namespaceGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}

	// Try to get the namespace
	ctx := context.Background()
	_, err := client.Resource(namespaceGVR).Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Try using oc command as a fallback
		_, ocErr := utils.RunCommand("oc", "get", "namespace", namespace)
		if ocErr != nil {
			return false, err
		}
		return true, nil
	}

	return true, nil
}

// getClusterMonitoringConfig gets the monitoring configuration from the cluster-monitoring-config ConfigMap
func getClusterMonitoringConfig(client dynamic.Interface) (bool, string) {
	// Define the ConfigMap resource
	configMapGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	// Try to get the cluster-monitoring-config ConfigMap
	ctx := context.Background()
	configMap, err := client.Resource(configMapGVR).Namespace("openshift-monitoring").Get(ctx, "cluster-monitoring-config", metav1.GetOptions{})
	if err != nil {
		// Try using oc command as a fallback
		configMapYaml, ocErr := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
		if ocErr != nil {
			return false, ""
		}
		return true, configMapYaml
	}

	// Convert to YAML for better readability
	configMapYaml, err := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
	if err != nil {
		// Extract the config.yaml data if oc command fails
		data, found, _ := unstructured.NestedMap(configMap.Object, "data")
		if !found {
			return true, "ConfigMap exists but data field not found"
		}

		configYaml, found := data["config.yaml"]
		if !found {
			return true, "ConfigMap exists but config.yaml not found"
		}

		return true, fmt.Sprintf("%v", configYaml)
	}

	return true, configMapYaml
}

// getUserWorkloadMonitoringConfig gets the configuration from the user-workload-monitoring-config ConfigMap
func getUserWorkloadMonitoringConfig(client dynamic.Interface) (bool, string) {
	// Define the ConfigMap resource
	configMapGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	// Try to get the user-workload-monitoring-config ConfigMap
	ctx := context.Background()
	configMap, err := client.Resource(configMapGVR).Namespace("openshift-user-workload-monitoring").Get(ctx, "user-workload-monitoring-config", metav1.GetOptions{})
	if err != nil {
		// Try using oc command as a fallback
		configMapYaml, ocErr := utils.RunCommand("oc", "get", "configmap", "user-workload-monitoring-config", "-n", "openshift-user-workload-monitoring", "-o", "yaml")
		if ocErr != nil {
			return false, ""
		}
		return true, configMapYaml
	}

	// Convert to YAML for better readability
	configMapYaml, err := utils.RunCommand("oc", "get", "configmap", "user-workload-monitoring-config", "-n", "openshift-user-workload-monitoring", "-o", "yaml")
	if err != nil {
		// Extract the config.yaml data if oc command fails
		data, found, _ := unstructured.NestedMap(configMap.Object, "data")
		if !found {
			return true, "ConfigMap exists but data field not found"
		}

		configYaml, found := data["config.yaml"]
		if !found {
			return true, "ConfigMap exists but config.yaml not found"
		}

		return true, fmt.Sprintf("%v", configYaml)
	}

	return true, configMapYaml
}

// checkUserWorkloadComponents checks which user workload monitoring components are configured
func checkUserWorkloadComponents(uwmConfigYaml string) ([]string, []string, string) {
	// Extract the actual config.yaml content
	configPattern := regexp.MustCompile(`(?s)config\.yaml:\s*\|(.*?)(?:kind:|$)`)
	configMatch := configPattern.FindStringSubmatch(uwmConfigYaml)

	var configYaml string
	if len(configMatch) >= 2 {
		configYaml = configMatch[1]
	} else {
		// Fall back to the original content if we can't extract config.yaml specifically
		configYaml = uwmConfigYaml
	}

	// Required components for user workload monitoring
	requiredComponents := []string{
		"prometheusOperator",
		"prometheus",
		"thanosRuler",
		"alertmanager",
	}

	// Track configured and missing components
	configuredComponents := []string{}
	missingComponents := []string{}

	// Check which components are explicitly configured in the ConfigMap
	for _, component := range requiredComponents {
		componentPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^[ \t]*%s:`, regexp.QuoteMeta(component)))
		if componentPattern.MatchString(configYaml) {
			configuredComponents = append(configuredComponents, component)
		} else {
			missingComponents = append(missingComponents, component)
		}
	}

	// Generate detailed output
	var details strings.Builder

	if len(configuredComponents) > 0 {
		details.WriteString("Components explicitly configured in user-workload-monitoring-config:\n\n")
		for _, component := range configuredComponents {
			details.WriteString(fmt.Sprintf("- %s\n", component))
		}
		details.WriteString("\n")
	} else {
		details.WriteString("WARNING: No components are explicitly configured in the user-workload-monitoring-config ConfigMap.\n")
		details.WriteString("All components are using default settings which may not be optimal for production.\n\n")
	}

	if len(missingComponents) > 0 {
		details.WriteString("WARNING: The following required components are NOT explicitly configured:\n\n")
		for _, component := range missingComponents {
			details.WriteString(fmt.Sprintf("- %s\n", component))
		}
		details.WriteString("\nThese components are running with default settings, which may not be optimal for your environment.\n")
		details.WriteString("Consider configuring these components explicitly for better control over resources and behavior.\n\n")
	}

	// Check if components are actually running
	details.WriteString("Verifying component status in the cluster:\n\n")

	for _, component := range requiredComponents {
		isRunning := isUserWorkloadComponentRunning(component)
		status := "✅ Running"
		if !isRunning {
			status = "❌ Not found or not ready"
		}
		details.WriteString(fmt.Sprintf("- %s: %s\n", component, status))
	}

	return configuredComponents, missingComponents, details.String()
}

// isUserWorkloadComponentRunning checks if a user workload monitoring component is running
func isUserWorkloadComponentRunning(component string) bool {
	// Map of components to their resource names to check
	resourceChecks := map[string]struct {
		resourceType string
		namespace    string
		name         string
	}{
		"alertmanager":       {"statefulset", "openshift-user-workload-monitoring", "alertmanager-user-workload"},
		"prometheus":         {"statefulset", "openshift-user-workload-monitoring", "prometheus-user-workload"},
		"thanosRuler":        {"statefulset", "openshift-user-workload-monitoring", "thanos-ruler-user-workload"},
		"prometheusOperator": {"deployment", "openshift-user-workload-monitoring", "prometheus-operator"},
	}

	check, found := resourceChecks[component]
	if !found {
		return false
	}

	// Run oc command to check if the resource exists
	out, err := utils.RunCommand("oc", "get", check.resourceType, check.name, "-n", check.namespace)
	return err == nil && strings.Contains(out, check.name)
}

// checkUserWorkloadComponentsRunning checks if all required user workload monitoring components are running
func checkUserWorkloadComponentsRunning() (bool, string) {
	// Required components for user workload monitoring
	requiredComponents := []string{
		"prometheusOperator",
		"prometheus",
		"thanosRuler",
		"alertmanager",
	}

	allRunning := true
	var details strings.Builder
	details.WriteString("User Workload Monitoring components status:\n\n")

	for _, component := range requiredComponents {
		isRunning := isUserWorkloadComponentRunning(component)
		status := "✅ Running"
		if !isRunning {
			status = "❌ Not found or not ready"
			allRunning = false
		}
		details.WriteString(fmt.Sprintf("- %s: %s\n", component, status))
	}

	return allRunning, details.String()
}

// checkUserWorkloadPersistentStorage checks if persistent storage is configured for user workload monitoring
func checkUserWorkloadPersistentStorage(uwmConfigYaml string) (bool, string) {
	// Check for volume claim template in the config
	hasVolumeClaimTemplate := strings.Contains(uwmConfigYaml, "volumeClaimTemplate")

	// Check if PVCs exist for user workload monitoring components
	pvcList, err := utils.RunCommand("oc", "get", "pvc", "-n", "openshift-user-workload-monitoring")

	var details strings.Builder
	if err != nil {
		details.WriteString("Failed to retrieve PVCs in openshift-user-workload-monitoring namespace\n\n")
	} else {
		details.WriteString("PVCs in openshift-user-workload-monitoring namespace:\n\n")
		details.WriteString("[source, bash]\n----\n")
		details.WriteString(pvcList)
		details.WriteString("\n----\n\n")
	}

	// Check for the presence of specific PVCs
	hasPrometheusPVC := strings.Contains(pvcList, "prometheus-user-workload")
	hasThanosRulerPVC := strings.Contains(pvcList, "thanos-ruler-user-workload")
	hasAlertmanagerPVC := strings.Contains(pvcList, "alertmanager-user-workload")

	if hasVolumeClaimTemplate {
		details.WriteString("Persistent storage is configured in the user workload monitoring config via volumeClaimTemplate\n\n")
	} else {
		details.WriteString("No volumeClaimTemplate found in the user workload monitoring config\n\n")
	}

	if hasPrometheusPVC {
		details.WriteString("Prometheus PVC found\n\n")
	}

	if hasThanosRulerPVC {
		details.WriteString("Thanos Ruler PVC found\n\n")
	}

	if hasAlertmanagerPVC {
		details.WriteString("Alertmanager PVC found\n\n")
	}

	return hasVolumeClaimTemplate || hasPrometheusPVC || hasThanosRulerPVC || hasAlertmanagerPVC, details.String()
}
