/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for monitoring component storage. It:

- Verifies if persistent storage is configured for monitoring components
- Examines the configuration of Prometheus and other monitoring tools
- Checks volume claim templates and storage configuration
- Provides recommendations for durable monitoring setup
- Helps ensure monitoring data persists across restarts

This check helps maintain reliable monitoring by ensuring monitoring data is stored persistently.
*/

package monitoring

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// PrometheusK8sConfig represents the Prometheus configuration in the ConfigMap
type PrometheusK8sConfig struct {
	Retention           string                 `yaml:"retention"`
	Resources           map[string]interface{} `yaml:"resources"`
	VolumeClaimTemplate map[string]interface{} `yaml:"volumeClaimTemplate"`
}

// ConfigData represents the structure of the cluster-monitoring-config ConfigMap
type ConfigData struct {
	EnableUserWorkload bool                `yaml:"enableUserWorkload"`
	PrometheusK8s      PrometheusK8sConfig `yaml:"prometheusK8s"`
}

// MonitoringStorageCheck checks if monitoring components have persistent storage configured
type MonitoringStorageCheck struct {
	healthcheck.BaseCheck
}

// NewMonitoringStorageCheck creates a new monitoring storage check
func NewMonitoringStorageCheck() *MonitoringStorageCheck {
	return &MonitoringStorageCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"monitoring-storage",
			"Monitoring Storage",
			"Checks if OpenShift monitoring components have persistent storage configured",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *MonitoringStorageCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Check if the openshift-monitoring namespace exists
	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), "openshift-monitoring", metav1.GetOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get openshift-monitoring namespace",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting openshift-monitoring namespace: %v", err)
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder

	// Add monitoring namespace information
	formattedDetailOut.WriteString("Monitoring Namespace: openshift-monitoring\n\n")

	// Check if the cluster-monitoring-config ConfigMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})

	var configMapExists bool
	var rawData string
	var configMapData string

	if err != nil {
		configMapExists = false
		configMapData = "Failed to get cluster-monitoring-config ConfigMap"
	} else {
		configMapExists = true

		// Check if the key exists
		data, ok := cm.Data["config.yaml"]
		if ok {
			rawData = data
			configMapData = fmt.Sprintf("ConfigMap cluster-monitoring-config exists with config.yaml:\n[source, yaml]\n----\n%s\n----\n\n", data)
		} else {
			configMapData = "ConfigMap cluster-monitoring-config exists but config.yaml is missing\n\n"
		}
	}

	formattedDetailOut.WriteString(configMapData)

	// Get detailed information for PVCs in the monitoring namespace
	pvcOut, err := utils.RunCommand("oc", "get", "pvc", "-n", "openshift-monitoring")
	if err == nil && strings.TrimSpace(pvcOut) != "" {
		formattedDetailOut.WriteString("Persistent Volume Claims in openshift-monitoring:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(pvcOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("No Persistent Volume Claims found in openshift-monitoring namespace\n\n")
	}

	// Check if monitoring components have persistent storage
	hasStorage := hasPrometheusK8sVolumeClaimTemplate(rawData)

	// Add storage configuration analysis
	if hasStorage {
		formattedDetailOut.WriteString("Storage Analysis: Persistent storage is configured for monitoring components\n\n")
	} else {
		formattedDetailOut.WriteString("Storage Analysis: No persistent storage configuration found for monitoring components\n\n")
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	if !configMapExists || !hasStorage {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift monitoring components do not have persistent storage configured",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure persistent storage for monitoring components")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/configuring-the-monitoring-stack", version))
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#configuring_persistent_storage_configuring-the-monitoring-stack", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Storage is properly configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift monitoring components have persistent storage configured",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}

// Helper function to check if Prometheus has volume claim template
func hasPrometheusK8sVolumeClaimTemplate(data string) bool {
	if data == "" {
		return false
	}

	var configData ConfigData
	if err := yaml.Unmarshal([]byte(data), &configData); err != nil {
		return false
	}

	return configData.PrometheusK8s.VolumeClaimTemplate != nil && len(configData.PrometheusK8s.VolumeClaimTemplate) > 0
}
