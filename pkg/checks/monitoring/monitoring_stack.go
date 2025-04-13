package monitoring

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// MonitoringStorageCheck checks if monitoring components have persistent storage configured
type MonitoringStorageCheck struct {
	healthcheck.BaseCheck
}

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

	// Check if the cluster-monitoring-config ConfigMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})

	var configMapExists bool
	var rawData string

	if err != nil {
		configMapExists = false
	} else {
		configMapExists = true

		// Check if the key exists
		data, ok := cm.Data["config.yaml"]
		if ok {
			rawData = data
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed monitoring ConfigMap information"
	}

	// Check if monitoring components have persistent storage
	hasStorage := hasPrometheusK8sVolumeClaimTemplate(rawData)

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

		result.Detail = detailedOut
		return result, nil
	}

	// Storage is properly configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift monitoring components have persistent storage configured",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}

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
			"Checks if monitoring for user-defined projects is enabled",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *UserWorkloadMonitoringCheck) Run() (healthcheck.Result, error) {
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

	// Check if the cluster-monitoring-config ConfigMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})

	var userWorkloadEnabled bool
	var configMapExists bool

	if err != nil {
		configMapExists = false
		userWorkloadEnabled = false
	} else {
		configMapExists = true

		// Check if the key exists
		data, ok := cm.Data["config.yaml"]
		if ok {
			var configData ConfigData
			if err := yaml.Unmarshal([]byte(data), &configData); err == nil {
				userWorkloadEnabled = configData.EnableUserWorkload
			}
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed monitoring ConfigMap information"
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	// Additional check: see if the user-workload-monitoring namespace exists
	_, nsErr := clientset.CoreV1().Namespaces().Get(context.TODO(), "openshift-user-workload-monitoring", metav1.GetOptions{})
	userWorkloadNamespaceExists := nsErr == nil

	// If both checks indicate user workload monitoring is enabled
	if (configMapExists && userWorkloadEnabled) || userWorkloadNamespaceExists {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"User workload monitoring is enabled",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// User workload monitoring is not enabled
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		"User workload monitoring is not enabled",
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Enable monitoring for user-defined projects")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/configuring-the-monitoring-stack", version))
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects", version))

	result.Detail = detailedOut
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
