/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for comprehensive monitoring stack configuration. It:

- Examines the entire monitoring stack configuration against OpenShift best practices
- Checks persistent storage configuration for monitoring components
- Verifies CPU and memory resource requests and limits for monitoring components
- Checks for proper component placement using node selectors or tolerations
- Examines retention time and size configuration for Prometheus
- Verifies remote write storage configuration for long-term metrics retention
- Provides comprehensive recommendations for optimal monitoring setup

This check helps administrators ensure the monitoring stack is properly configured for reliability and performance.
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

// MonitoringStackConfigCheck checks if the monitoring stack is properly configured
type MonitoringStackConfigCheck struct {
	healthcheck.BaseCheck
}

// NewMonitoringStackConfigCheck creates a new monitoring stack configuration check
func NewMonitoringStackConfigCheck() *MonitoringStackConfigCheck {
	return &MonitoringStackConfigCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"monitoring-stack-config",
			"Monitoring Stack Configuration",
			"Checks if the monitoring stack is properly configured according to best practices",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *MonitoringStackConfigCheck) Run() (healthcheck.Result, error) {
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

	// Get the cluster-monitoring-config ConfigMap
	monConfigExists, monConfigYaml := getMonitoringConfig(client)

	// Parse node count to determine if this is a multi-node or single-node cluster
	isMultiNode, nodeCount := isMultiNodeCluster()

	// Initialize detailed output for the report
	var detailedOut strings.Builder
	detailedOut.WriteString("== Monitoring Stack Configuration Analysis ==\n\n")

	// Init a list to track issues
	var issues []string
	var recommendations []string

	// Add monitoring config to the detailed output
	if monConfigExists {
		detailedOut.WriteString("Cluster Monitoring Config exists:\n")
		detailedOut.WriteString(monConfigYaml)
		detailedOut.WriteString("\n\n")
	} else {
		detailedOut.WriteString("No custom cluster-monitoring-config ConfigMap found. Using default configuration.\n\n")
		issues = append(issues, "no custom monitoring configuration found, using defaults")
		recommendations = append(recommendations, "Create a cluster-monitoring-config ConfigMap in the openshift-monitoring namespace to customize the monitoring stack")
	}

	// Check which monitoring components are deployed or missing
	configuredComponents, unconfiguredComponents, componentDetails := checkMonitoringComponents(monConfigYaml)
	detailedOut.WriteString(componentDetails)
	detailedOut.WriteString("\n\n")

	if len(unconfiguredComponents) > 0 {
		issues = append(issues, fmt.Sprintf("monitoring stack is missing configuration for components: %s", strings.Join(unconfiguredComponents, ", ")))
		recommendations = append(recommendations, "Configure all recommended monitoring stack components for comprehensive monitoring")
	}

	// Note: User workload monitoring is checked by the dedicated UserWorkloadMonitoringCheck
	// This avoids duplication and follows the single responsibility principle

	// Check for persistent storage configuration
	hasPVC, pvcDetails := checkPersistentStorage(client, monConfigYaml)
	detailedOut.WriteString("== Persistent Storage Configuration ==\n")
	detailedOut.WriteString(pvcDetails)
	detailedOut.WriteString("\n\n")

	if !hasPVC && isMultiNode {
		issues = append(issues, "persistent storage not configured for monitoring components in a multi-node cluster")
		recommendations = append(recommendations, "Configure persistent storage for Prometheus, Alertmanager, and Thanos Ruler to ensure high availability and data persistence")
	}

	// Check for resource requests and limits
	hasResourceLimits, resourceDetails := checkResourceLimits(monConfigYaml)
	detailedOut.WriteString("== Resource Requests and Limits ==\n")
	detailedOut.WriteString(resourceDetails)
	detailedOut.WriteString("\n\n")

	if !hasResourceLimits {
		issues = append(issues, "resource requests and limits not explicitly configured for monitoring components")
		recommendations = append(recommendations, "Configure CPU and memory resource requests and limits for monitoring components to ensure stability")
	}

	// Check for node placement configuration
	hasNodePlacement, nodeDetails, missingPlacementComponents := checkNodePlacement(monConfigYaml)
	detailedOut.WriteString("== Node Placement Configuration ==\n")
	detailedOut.WriteString(nodeDetails)
	detailedOut.WriteString("\n\n")

	if !hasNodePlacement && nodeCount > 3 {
		if len(missingPlacementComponents) > 0 {
			issues = append(issues, fmt.Sprintf("node placement configuration incomplete for components: %s", strings.Join(missingPlacementComponents, ", ")))
		} else {
			issues = append(issues, "node placement configuration incomplete for monitoring components")
		}
		recommendations = append(recommendations, "Configure nodeSelector and tolerations for all monitoring components to place them on infrastructure nodes")
	}

	// Check for retention time and size configuration
	hasRetentionConfig, retentionDetails := checkRetentionConfig(monConfigYaml)
	detailedOut.WriteString("== Data Retention Configuration ==\n")
	detailedOut.WriteString(retentionDetails)
	detailedOut.WriteString("\n\n")

	if !hasRetentionConfig {
		issues = append(issues, "no custom retention time or size configured for Prometheus metrics")
		recommendations = append(recommendations, "Configure retention time and size for Prometheus metrics to manage disk space usage")
	}

	// Check for remote write configuration
	hasRemoteWrite, remoteWriteDetails := checkRemoteWriteConfig(monConfigYaml)
	detailedOut.WriteString("== Remote Write Configuration ==\n")
	detailedOut.WriteString(remoteWriteDetails)
	detailedOut.WriteString("\n\n")

	if !hasRemoteWrite {
		issues = append(issues, "remote write storage not configured for long-term metrics retention")
		recommendations = append(recommendations, "Configure remote write storage for long-term metrics retention and historical analysis")
	}

	// Check for alerting configuration
	hasAlertRouting, alertDetails := checkAlertRouting()
	detailedOut.WriteString("== Alert Routing Configuration ==\n")
	detailedOut.WriteString(alertDetails)
	detailedOut.WriteString("\n\n")

	if !hasAlertRouting {
		issues = append(issues, "no custom alert routing configuration found")
		recommendations = append(recommendations, "Configure alert routing to ensure alerts are properly delivered to the right teams")
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.14" // Default to a known version if we can't determine
	}

	// Generate recommendations based on documented best practices
	detailedOut.WriteString("== Best Practices Recommendations ==\n")
	detailedOut.WriteString("1. For production clusters, configure persistent storage for monitoring components\n")
	detailedOut.WriteString("2. Set appropriate CPU and memory limits for monitoring components\n")
	detailedOut.WriteString("3. Configure node placement to isolate monitoring components on dedicated nodes\n")
	detailedOut.WriteString("4. Set appropriate retention time and size for Prometheus data\n")
	detailedOut.WriteString("5. Configure remote write for long-term metrics storage\n")
	detailedOut.WriteString("6. Set up alert routing to ensure notifications reach the right people\n")
	detailedOut.WriteString("7. Enable user workload monitoring for application metrics\n")
	detailedOut.WriteString("\n")

	// Add sample configuration section
	detailedOut.WriteString("== Sample Configuration ==\n\n")
	detailedOut.WriteString("Below is a sample configuration for the cluster-monitoring-config ConfigMap that includes proper resource limits, storage configuration, and node placement:\n\n")
	detailedOut.WriteString("[source,yaml]\n")
	detailedOut.WriteString("----\n")
	detailedOut.WriteString("apiVersion: v1\n")
	detailedOut.WriteString("kind: ConfigMap\n")
	detailedOut.WriteString("metadata:\n")
	detailedOut.WriteString("  name: cluster-monitoring-config\n")
	detailedOut.WriteString("  namespace: openshift-monitoring\n")
	detailedOut.WriteString("data:\n")
	detailedOut.WriteString("  config.yaml: |\n")
	detailedOut.WriteString("    enableUserWorkload: true\n")
	detailedOut.WriteString("    prometheusOperator:\n")
	detailedOut.WriteString("      resources:\n")
	detailedOut.WriteString("        limits:\n")
	detailedOut.WriteString("          cpu: 500m\n")
	detailedOut.WriteString("          memory: 1Gi\n")
	detailedOut.WriteString("        requests:\n")
	detailedOut.WriteString("          cpu: 200m\n")
	detailedOut.WriteString("          memory: 500Mi\n")
	detailedOut.WriteString("      nodeSelector:\n")
	detailedOut.WriteString("        node-role.kubernetes.io/infra: \"\"\n")
	detailedOut.WriteString("      tolerations:\n")
	detailedOut.WriteString("      - key: node-role.kubernetes.io/infra\n")
	detailedOut.WriteString("        effect: NoSchedule\n")
	detailedOut.WriteString("    prometheusK8s:\n")
	detailedOut.WriteString("      resources:\n")
	detailedOut.WriteString("        limits:\n")
	detailedOut.WriteString("          cpu: 500m\n")
	detailedOut.WriteString("          memory: 3Gi\n")
	detailedOut.WriteString("        requests:\n")
	detailedOut.WriteString("          cpu: 200m\n")
	detailedOut.WriteString("          memory: 500Mi\n")
	detailedOut.WriteString("      retention: 7d\n")
	detailedOut.WriteString("      volumeClaimTemplate:\n")
	detailedOut.WriteString("        spec:\n")
	detailedOut.WriteString("          storageClassName: gp3-csi\n")
	detailedOut.WriteString("          resources:\n")
	detailedOut.WriteString("            requests:\n")
	detailedOut.WriteString("              storage: 40Gi\n")
	detailedOut.WriteString("      nodeSelector:\n")
	detailedOut.WriteString("        node-role.kubernetes.io/infra: \"\"\n")
	detailedOut.WriteString("      tolerations:\n")
	detailedOut.WriteString("      - key: node-role.kubernetes.io/infra\n")
	detailedOut.WriteString("        effect: NoSchedule\n")
	detailedOut.WriteString("----\n")
	detailedOut.WriteString("\nNote: The above is a simplified example. You should adjust resource limits, storage size, and other parameters based on your cluster size and workload.\n")

	// If there are no issues, return OK
	if len(issues) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Monitoring stack is configured according to best practices",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut.String()
		return result, nil
	}

	// Create a result based on the number and severity of issues
	status := types.StatusWarning
	resultKey := types.ResultKeyRecommended

	// Create a descriptive message
	message := fmt.Sprintf("Monitoring stack configuration needs improvement: %s", strings.Join(issues[:minInt(3, len(issues))], ", "))

	result := healthcheck.NewResult(
		c.ID(),
		status,
		message,
		resultKey,
	)

	// Add recommendations
	for _, rec := range recommendations {
		result.AddRecommendation(rec)
	}

	// Add documentation link
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index", version))

	result.Detail = detailedOut.String()
	return result, nil
}

// checkMonitoringComponents checks which monitoring components are configured in the cluster-monitoring-config ConfigMap
func checkMonitoringComponents(monConfigYaml string) ([]string, []string, string) {
	// Extract the actual config.yaml content
	configPattern := regexp.MustCompile(`(?s)config\.yaml:\s*\|(.*?)(?:kind:|$)`)
	configMatch := configPattern.FindStringSubmatch(monConfigYaml)

	var configYaml string
	if len(configMatch) >= 2 {
		configYaml = configMatch[1]
	} else {
		// Fall back to the original content if we can't extract config.yaml specifically
		configYaml = monConfigYaml
	}

	// List of configurable core platform monitoring components per OpenShift documentation
	configurableComponents := []string{
		"prometheusOperator",
		"prometheusK8s",
		"alertmanagerMain",
		"thanosQuerier",
		"kubeStateMetrics",
		"monitoringPlugin",
		"openshiftStateMetrics",
		"telemeterClient",
		"metricsServer",
		"k8sPrometheusAdapter", // Added from the sample config
	}

	// Additional components that may be configured
	additionalComponents := []string{
		"nodeExporter",
		"prometheusOperatorAdmissionWebhook",
		"thanosRuler",
	}

	// Combine all expected components
	allComponents := append(configurableComponents, additionalComponents...)

	// Track configured and unconfigured components
	configuredComponents := []string{}
	unconfiguredComponents := []string{}

	// Check which components are explicitly configured in the ConfigMap
	for _, component := range allComponents {
		componentPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^[ \t]*%s:`, regexp.QuoteMeta(component)))
		if componentPattern.MatchString(configYaml) {
			configuredComponents = append(configuredComponents, component)
		} else {
			// Only add to unconfigured list if it's in the core configurable components
			for _, coreComponent := range configurableComponents {
				if component == coreComponent {
					unconfiguredComponents = append(unconfiguredComponents, component)
					break
				}
			}
		}
	}

	// Check if user workload monitoring is enabled
	uwmEnabled := strings.Contains(configYaml, "enableUserWorkload: true")

	// Generate detailed output
	var details strings.Builder

	details.WriteString("== Monitoring Stack Components ==\n\n")

	if len(configuredComponents) > 0 {
		details.WriteString("Components explicitly configured in cluster-monitoring-config:\n")
		for _, component := range configuredComponents {
			details.WriteString(fmt.Sprintf("- %s\n", component))
		}
		details.WriteString("\n")
	} else {
		details.WriteString("WARNING: No monitoring components are explicitly configured in the cluster-monitoring-config ConfigMap.\n")
		details.WriteString("All components are using default settings which may not be optimal for production.\n\n")
	}

	if len(unconfiguredComponents) > 0 {
		details.WriteString("WARNING: The following core components are NOT explicitly configured in cluster-monitoring-config:\n")
		for _, component := range unconfiguredComponents {
			details.WriteString(fmt.Sprintf("- %s\n", component))
		}
		details.WriteString("\nThese components are running with default settings, which may not be optimal for your environment.\n")
		details.WriteString("Consider configuring these components explicitly for better control over resources and behavior.\n\n")
	}

	details.WriteString(fmt.Sprintf("User Workload Monitoring enabled: %v\n", uwmEnabled))
	if !uwmEnabled {
		details.WriteString("NOTE: User Workload Monitoring is not enabled. Consider enabling it to monitor application workloads.\n")
	}

	// Check if components are actually running
	details.WriteString("\nVerifying component status in the cluster:\n")

	for _, component := range allComponents {
		isRunning := isComponentRunning(component)
		status := "✅ Running"
		if !isRunning {
			status = "❌ Not found or not ready"
		}
		details.WriteString(fmt.Sprintf("- %s: %s\n", component, status))
	}

	return configuredComponents, unconfiguredComponents, details.String()
}

// isComponentRunning checks if a monitoring component is actually running in the cluster
func isComponentRunning(component string) bool {
	// Map of components to their resource names to check
	resourceChecks := map[string]struct {
		resourceType string
		namespace    string
		name         string
	}{
		"alertmanagerMain":                   {"statefulset", "openshift-monitoring", "alertmanager-main"},
		"prometheusK8s":                      {"statefulset", "openshift-monitoring", "prometheus-k8s"},
		"thanosQuerier":                      {"deployment", "openshift-monitoring", "thanos-querier"},
		"prometheusOperator":                 {"deployment", "openshift-monitoring", "prometheus-operator"},
		"metricsServer":                      {"deployment", "openshift-monitoring", "metrics-server"},
		"kubeStateMetrics":                   {"deployment", "openshift-monitoring", "kube-state-metrics"},
		"telemeterClient":                    {"deployment", "openshift-monitoring", "telemeter-client"},
		"openshiftStateMetrics":              {"deployment", "openshift-monitoring", "openshift-state-metrics"},
		"nodeExporter":                       {"daemonset", "openshift-monitoring", "node-exporter"},
		"monitoringPlugin":                   {"deployment", "openshift-monitoring", "monitoring-plugin"},
		"prometheusOperatorAdmissionWebhook": {"deployment", "openshift-monitoring", "prometheus-operator-admission-webhook"},
		"thanosRuler":                        {"statefulset", "openshift-monitoring", "thanos-ruler"},
		"k8sPrometheusAdapter":               {"deployment", "openshift-monitoring", "prometheus-adapter"},
	}

	check, found := resourceChecks[component]
	if !found {
		return false
	}

	// Run oc command to check if the resource exists
	out, err := utils.RunCommand("oc", "get", check.resourceType, check.name, "-n", check.namespace)
	return err == nil && strings.Contains(out, check.name)
}

// getMonitoringConfig gets the monitoring configuration from the cluster-monitoring-config ConfigMap
func getMonitoringConfig(client dynamic.Interface) (bool, string) {
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

// isMultiNodeCluster determines if the cluster has multiple nodes
func isMultiNodeCluster() (bool, int) {
	nodeList, err := utils.RunCommand("oc", "get", "nodes", "-o", "name")
	if err != nil {
		return false, 0
	}

	nodes := strings.Split(strings.TrimSpace(nodeList), "\n")
	return len(nodes) > 1, len(nodes)
}

// checkPersistentStorage checks if persistent storage is configured for monitoring components
func checkPersistentStorage(client dynamic.Interface, monConfigYaml string) (bool, string) {
	// Check for volume claim template in the config
	hasVolumeClaimTemplate := strings.Contains(monConfigYaml, "volumeClaimTemplate")

	// Check if PVCs exist for monitoring components
	pvcList, err := utils.RunCommand("oc", "get", "pvc", "-n", "openshift-monitoring")

	var details strings.Builder
	if err != nil {
		details.WriteString("Failed to retrieve PVCs in openshift-monitoring namespace\n")
	} else {
		details.WriteString("PVCs in openshift-monitoring namespace:\n")
		details.WriteString(pvcList)
		details.WriteString("\n")
	}

	// Check for the presence of specific PVCs
	hasPrometheusPVC := strings.Contains(pvcList, "prometheus-k8s")
	hasAlertmanagerPVC := strings.Contains(pvcList, "alertmanager-main")
	hasThanosRulerPVC := strings.Contains(pvcList, "thanos-ruler")

	if hasVolumeClaimTemplate {
		details.WriteString("Persistent storage is configured in the monitoring config via volumeClaimTemplate\n")
	} else {
		details.WriteString("No volumeClaimTemplate found in the monitoring config\n")
	}

	if hasPrometheusPVC {
		details.WriteString("Prometheus PVC found\n")
	}

	if hasAlertmanagerPVC {
		details.WriteString("Alertmanager PVC found\n")
	}

	if hasThanosRulerPVC {
		details.WriteString("Thanos Ruler PVC found\n")
	}

	return hasVolumeClaimTemplate || hasPrometheusPVC || hasAlertmanagerPVC || hasThanosRulerPVC, details.String()
}

// checkResourceLimits checks if resource requests and limits are configured for monitoring components
func checkResourceLimits(monConfigYaml string) (bool, string) {
	// Check for resources configuration in the config
	hasResourceLimits := strings.Contains(monConfigYaml, "resources") &&
		(strings.Contains(monConfigYaml, "limits") || strings.Contains(monConfigYaml, "requests"))

	var details strings.Builder
	if hasResourceLimits {
		details.WriteString("Resource requests and/or limits are configured in the monitoring config\n")

		// Try to extract resource configurations for key components
		resourceConfigs := regexp.MustCompile(`(?s)(prometheus|alertmanager|thanosQuerier)K?8?s?:\s*\n\s*resources:\s*\n([\s\S]*?)(?:\n\w|\z)`).FindAllStringSubmatch(monConfigYaml, -1)

		if len(resourceConfigs) > 0 {
			for _, match := range resourceConfigs {
				if len(match) >= 3 {
					details.WriteString(fmt.Sprintf("\nResource configuration for %s:\n%s\n", match[1], match[2]))
				}
			}
		}
	} else {
		details.WriteString("No explicit resource requests and limits found in the monitoring config\n")
		details.WriteString("Using default resource configurations\n")
	}

	return hasResourceLimits, details.String()
}

// checkNodePlacement checks if node placement is configured for monitoring components
func checkNodePlacement(monConfigYaml string) (bool, string, []string) {
	// Extract the actual config.yaml content
	configPattern := regexp.MustCompile(`(?s)config\.yaml:\s*\|(.*?)(?:kind:|$)`)
	configMatch := configPattern.FindStringSubmatch(monConfigYaml)

	var configYaml string
	if len(configMatch) >= 2 {
		configYaml = configMatch[1]
	} else {
		// Fall back to the original content if we can't extract config.yaml specifically
		configYaml = monConfigYaml
	}

	// Simple direct check for each component by name
	componentsWithNodeSelector := make(map[string]bool)
	componentsWithTolerations := make(map[string]bool)

	// List of monitoring components to check
	monitoringComponents := []string{
		"prometheusOperator",
		"prometheusK8s",
		"alertmanagerMain",
		"thanosQuerier",
		"kubeStateMetrics",
		"monitoringPlugin",
		"openshiftStateMetrics",
		"telemeterClient",
		"k8sPrometheusAdapter",
	}

	// Check each component directly in the yaml
	for _, component := range monitoringComponents {
		// Check for the component section in the yaml
		componentPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^[ \t]*%s:`, regexp.QuoteMeta(component)))
		if componentPattern.MatchString(configYaml) {
			// Extract the component's configuration block
			componentBlockPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^[ \t]*%s:.*?\n((?:[ \t]+.*\n)*)`, regexp.QuoteMeta(component)))
			componentMatches := componentBlockPattern.FindStringSubmatch(configYaml)

			if len(componentMatches) >= 2 {
				componentBlock := componentMatches[1]

				// Check for nodeSelector
				if strings.Contains(componentBlock, "nodeSelector:") {
					componentsWithNodeSelector[component] = true
				}

				// Check for tolerations
				if strings.Contains(componentBlock, "tolerations:") {
					componentsWithTolerations[component] = true
				}
			}
		}
	}

	var details strings.Builder

	// Check each component for proper node placement configuration
	var missingPlacementComponents []string

	for _, component := range monitoringComponents {
		// Check if component exists in config (we look for the component name followed by a colon)
		componentPattern := regexp.MustCompile(fmt.Sprintf(`(?m)^[ \t]*%s:`, regexp.QuoteMeta(component)))
		componentExists := componentPattern.MatchString(configYaml)

		if !componentExists {
			details.WriteString(fmt.Sprintf("Component %s is not configured in the monitoring config\n", component))
			missingPlacementComponents = append(missingPlacementComponents, component)
			continue
		}

		hasNodeSelector := componentsWithNodeSelector[component]
		hasTolerations := componentsWithTolerations[component]

		details.WriteString(fmt.Sprintf("Component %s has nodeSelector: %v, tolerations: %v\n",
			component, hasNodeSelector, hasTolerations))

		if !hasNodeSelector || !hasTolerations {
			missingPlacementComponents = append(missingPlacementComponents, component)
		}
	}

	// Add summary information about missing configurations
	if len(missingPlacementComponents) > 0 {
		details.WriteString(fmt.Sprintf("\nWARNING: Missing node placement configuration for components: %s\n",
			strings.Join(missingPlacementComponents, ", ")))
	} else {
		details.WriteString("\nAll monitoring components have proper node placement configuration\n")
	}

	// Check for infrastructure nodes
	infraNodesOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=")
	hasInfraNodes := err == nil && !strings.Contains(infraNodesOut, "No resources found")

	if hasInfraNodes {
		details.WriteString("\nInfrastructure nodes found in the cluster\n")

		// Get monitoring pods
		monPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-monitoring", "-o", "wide")
		if err == nil && monPodsOut != "" {
			// Get infra node names
			infraNodeNames, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "jsonpath={.items[*].metadata.name}")
			if err == nil && infraNodeNames != "" {
				// Create a map for faster lookup
				infraNodes := make(map[string]bool)
				for _, nodeName := range strings.Split(infraNodeNames, " ") {
					if nodeName != "" {
						infraNodes[nodeName] = true
					}
				}

				// Check if monitoring pods are running on infra nodes
				monPodLines := strings.Split(monPodsOut, "\n")
				podsOnInfraNodes := false
				if len(monPodLines) > 1 { // Skip header line
					for _, line := range monPodLines[1:] {
						if line == "" {
							continue
						}
						fields := strings.Fields(line)
						if len(fields) >= 7 { // Check if the line has enough fields
							podName := fields[0]
							nodeName := fields[6]
							// Check for monitoring components and if they're on infra nodes
							if (strings.Contains(podName, "prometheus") ||
								strings.Contains(podName, "alertmanager") ||
								strings.Contains(podName, "thanos") ||
								strings.Contains(podName, "state-metrics") ||
								strings.Contains(podName, "telemeter")) && infraNodes[nodeName] {
								podsOnInfraNodes = true
								break
							}
						}
					}
				}

				if podsOnInfraNodes {
					details.WriteString("\nMonitoring pods are running on infrastructure nodes\n")
				} else {
					details.WriteString("\nNo monitoring pods found running on infrastructure nodes\n")
				}
			} else {
				details.WriteString("\nCould not determine infrastructure node names\n")
			}
		} else {
			details.WriteString("\nCould not get information about monitoring pods\n")
		}
	} else {
		details.WriteString("\nNo infrastructure nodes found in the cluster\n")
	}

	return len(missingPlacementComponents) == 0, details.String(), missingPlacementComponents
}

// checkRetentionConfig checks if retention time and size are configured for Prometheus
func checkRetentionConfig(monConfigYaml string) (bool, string) {
	// Check for retention or retentionSize in the config
	hasRetention := strings.Contains(monConfigYaml, "retention:")
	hasRetentionSize := strings.Contains(monConfigYaml, "retentionSize:")

	var details strings.Builder
	if hasRetention || hasRetentionSize {
		details.WriteString("Prometheus data retention is configured\n")

		// Try to extract retention configurations
		retentionConfig := regexp.MustCompile(`(?s)prometheusK8s:\s*\n(.*?)(retention: [^\n]*)`).FindStringSubmatch(monConfigYaml)
		if len(retentionConfig) >= 3 {
			details.WriteString(fmt.Sprintf("\nRetention time: %s\n", retentionConfig[2]))
		} else {
			details.WriteString("\nUsing default retention time (15d)\n")
		}

		retentionSizeConfig := regexp.MustCompile(`(?s)prometheusK8s:\s*\n(.*?)(retentionSize: [^\n]*)`).FindStringSubmatch(monConfigYaml)
		if len(retentionSizeConfig) >= 3 {
			details.WriteString(fmt.Sprintf("\nRetention size: %s\n", retentionSizeConfig[2]))
		} else {
			details.WriteString("\nNo retention size limit configured\n")
		}
	} else {
		details.WriteString("Using default Prometheus retention configuration (15d with no size limit)\n")
	}

	return hasRetention || hasRetentionSize, details.String()
}

// checkRemoteWriteConfig checks if remote write is configured for Prometheus
func checkRemoteWriteConfig(monConfigYaml string) (bool, string) {
	// Check for remoteWrite in the config
	hasRemoteWrite := strings.Contains(monConfigYaml, "remoteWrite:")

	var details strings.Builder
	if hasRemoteWrite {
		details.WriteString("Remote write is configured for Prometheus\n")

		// Try to extract remote write configurations
		remoteWriteConfig := regexp.MustCompile(`(?s)remoteWrite:\s*\n([\s\S]*?)(?:\n\w|\z)`).FindStringSubmatch(monConfigYaml)
		if len(remoteWriteConfig) >= 2 {
			details.WriteString("\nRemote write configuration:\n")
			details.WriteString(remoteWriteConfig[1])
		}

		// Check for additional details like URLs (with sensitive info redacted)
		urlConfig := regexp.MustCompile(`url: "([^"]*)`).FindStringSubmatch(monConfigYaml)
		if len(urlConfig) >= 2 {
			details.WriteString(fmt.Sprintf("\nRemote write endpoint: %s\n", urlConfig[1]))
		}
	} else {
		details.WriteString("No remote write configuration found\n")
		details.WriteString("Consider configuring remote write for long-term metrics storage\n")
	}

	return hasRemoteWrite, details.String()
}

// checkAlertRouting checks if alert routing is configured
func checkAlertRouting() (bool, string) {
	// Get the alertmanager-main secret - first get the encoded data
	encodedOut, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring", "-o", "jsonpath={.data.alertmanager\\.yaml}")
	if err != nil {
		return false, "Failed to get Alertmanager configuration: " + err.Error()
	}

	// Now decode the base64 data - using bash to decode
	out, err := utils.RunCommandWithInput(encodedOut, "base64", "--decode")
	if err != nil {
		return false, "Failed to decode Alertmanager configuration: " + err.Error()
	}

	// Check if there are receivers configured other than "default" and "watchdog"
	hasCustomReceivers := strings.Contains(out, "name: default") &&
		strings.Contains(out, "receivers:") &&
		!strings.Contains(out, "receivers:\n- name: default\n- name: watchdog")

	// Check if specific receiver types are configured
	hasReceiverConfigs := strings.Contains(out, "_configs") ||
		strings.Contains(out, "configs:")

	var details strings.Builder
	details.WriteString("Alertmanager configuration:\n")
	details.WriteString(out)
	details.WriteString("\n\n")

	if hasCustomReceivers {
		details.WriteString("Custom alert receivers are configured\n")

		// Try to identify receiver types
		if strings.Contains(out, "pagerduty_configs") {
			details.WriteString("- PagerDuty integration configured\n")
		}

		if strings.Contains(out, "slack_configs") {
			details.WriteString("- Slack integration configured\n")
		}

		if strings.Contains(out, "email_configs") {
			details.WriteString("- Email integration configured\n")
		}

		if strings.Contains(out, "webhook_configs") {
			details.WriteString("- Webhook integration configured\n")
		}
	} else {
		details.WriteString("No custom alert receivers found beyond default configuration\n")
		details.WriteString("Configure alert receivers to ensure notifications are sent to the right teams\n")
	}

	if hasReceiverConfigs {
		details.WriteString("\nAlert receiver configurations found\n")
	} else {
		details.WriteString("\nNo specific alert receiver configurations found\n")
	}

	return hasCustomReceivers && hasReceiverConfigs, details.String()
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
