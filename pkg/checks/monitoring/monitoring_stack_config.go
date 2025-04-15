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
	hasNodePlacement, placementDetails := checkNodePlacement(monConfigYaml)
	detailedOut.WriteString("== Node Placement Configuration ==\n")
	detailedOut.WriteString(placementDetails)
	detailedOut.WriteString("\n\n")

	if !hasNodePlacement && nodeCount > 3 {
		issues = append(issues, "monitoring components not configured for specific node placement in a cluster with multiple nodes")
		recommendations = append(recommendations, "Configure nodeSelector or tolerations to place monitoring components on dedicated infrastructure nodes")
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
	detailedOut.WriteString("\n")

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
func checkNodePlacement(monConfigYaml string) (bool, string) {
	// Check for nodeSelector or tolerations in the config
	hasNodeSelector := strings.Contains(monConfigYaml, "nodeSelector")
	hasTolerations := strings.Contains(monConfigYaml, "tolerations")

	var details strings.Builder
	if hasNodeSelector {
		details.WriteString("NodeSelector is configured for monitoring components\n")

		// Try to extract nodeSelector configurations
		nodeSelectorConfigs := regexp.MustCompile(`(?s)(prometheus|alertmanager|thanosQuerier)K?8?s?:\s*\n\s*nodeSelector:\s*\n([\s\S]*?)(?:\n\w|\z)`).FindAllStringSubmatch(monConfigYaml, -1)

		if len(nodeSelectorConfigs) > 0 {
			for _, match := range nodeSelectorConfigs {
				if len(match) >= 3 {
					details.WriteString(fmt.Sprintf("\nNodeSelector configuration for %s:\n%s\n", match[1], match[2]))
				}
			}
		}
	} else {
		details.WriteString("No nodeSelector configuration found\n")
	}

	if hasTolerations {
		details.WriteString("\nTolerations are configured for monitoring components\n")

		// Try to extract tolerations configurations
		tolerationConfigs := regexp.MustCompile(`(?s)(prometheus|alertmanager|thanosQuerier)K?8?s?:\s*\n\s*tolerations:\s*\n([\s\S]*?)(?:\n\w|\z)`).FindAllStringSubmatch(monConfigYaml, -1)

		if len(tolerationConfigs) > 0 {
			for _, match := range tolerationConfigs {
				if len(match) >= 3 {
					details.WriteString(fmt.Sprintf("\nTolerations configuration for %s:\n%s\n", match[1], match[2]))
				}
			}
		}
	} else {
		details.WriteString("\nNo tolerations configuration found\n")
	}

	// Check if monitoring components are running on infra nodes
	infraNodesOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=")
	hasInfraNodes := err == nil && !strings.Contains(infraNodesOut, "No resources found")

	if hasInfraNodes {
		details.WriteString("\nInfrastructure nodes found in the cluster\n")

		// Check if monitoring pods are running on infra nodes
		monPodsOnInfraNodes, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-monitoring", "-o", "wide", "|", "grep", "-E", "prometheus|alertmanager|thanos", "|", "grep", "-E", "$(oc get nodes -l node-role.kubernetes.io/infra= -o name | cut -d/ -f2 | paste -sd\"|\" -)")

		if err == nil && monPodsOnInfraNodes != "" {
			details.WriteString("\nMonitoring pods are running on infrastructure nodes\n")
		} else {
			details.WriteString("\nNo monitoring pods found running on infrastructure nodes\n")
		}
	} else {
		details.WriteString("\nNo infrastructure nodes found in the cluster\n")
	}

	return hasNodeSelector || hasTolerations, details.String()
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
	// Get the alertmanager-main secret
	out, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring", "-o", "jsonpath={.data.alertmanager\\.yaml}", "|", "base64", "-d")
	if err != nil {
		return false, "Failed to get Alertmanager configuration"
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
