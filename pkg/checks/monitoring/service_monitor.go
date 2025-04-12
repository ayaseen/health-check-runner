package monitoring

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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
			healthcheck.CategoryMonitoring,
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
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client configuration",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client configuration: %v", err)
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
		// Check if user workload monitoring is enabled
		cmExists, _ := checkUserWorkloadMonitoringEnabled()
		if !cmExists {
			result := healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusWarning,
				"User Workload Monitoring is not enabled in the cluster",
				healthcheck.ResultKeyRecommended,
			)

			// Get OpenShift version for documentation links
			version, verErr := utils.GetOpenShiftMajorMinorVersion()
			if verErr != nil {
				version = "4.10" // Default to a known version if we can't determine
			}

			result.AddRecommendation("Enable User Workload Monitoring to monitor application metrics")
			result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects", version))

			return result, nil
		}

		// If the user workload monitoring is enabled but we can't get ServiceMonitors, there's a different issue
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to list ServiceMonitors",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error listing ServiceMonitors: %v", err)
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

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Generate detailed output for the report
	var detailedOut strings.Builder
	detailedOut.WriteString("ServiceMonitors found:\n\n")

	if len(userServiceMonitors) > 0 {
		for _, sm := range userServiceMonitors {
			detailedOut.WriteString(fmt.Sprintf("Namespace: %s, Name: %s\n", sm.GetNamespace(), sm.GetName()))
		}
	} else {
		detailedOut.WriteString("No user ServiceMonitors found\n")
	}

	// Check if there are any user ServiceMonitors
	if len(userServiceMonitors) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No ServiceMonitors found for application metrics monitoring",
			healthcheck.ResultKeyRecommended,
		)

		result.AddRecommendation("Create ServiceMonitors for your applications to collect custom metrics")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#specifying-how-a-service-is-monitored", version))

		result.Detail = detailedOut.String()
		return result, nil
	}

	// At least one user ServiceMonitor exists
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Found %d ServiceMonitors for application metrics monitoring", len(userServiceMonitors)),
		healthcheck.ResultKeyNoChange,
	)

	result.Detail = detailedOut.String()
	return result, nil
}

// checkUserWorkloadMonitoringEnabled checks if the user workload monitoring is enabled
func checkUserWorkloadMonitoringEnabled() (bool, error) {
	// Check if the openshift-user-workload-monitoring namespace exists
	out, err := utils.RunCommand("oc", "get", "namespace", "openshift-user-workload-monitoring")
	if err != nil {
		return false, err
	}

	if strings.Contains(out, "openshift-user-workload-monitoring") {
		return true, nil
	}

	// Check the cluster-monitoring-config ConfigMap for enableUserWorkload setting
	out, err = utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "jsonpath={.data.config\\.yaml}")
	if err != nil {
		return false, err
	}

	if strings.Contains(out, "enableUserWorkload: true") {
		return true, nil
	}

	return false, nil
}
