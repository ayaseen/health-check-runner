/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06

This application performs health checks on OpenShift to provide visibility into various functionalities. It verifies the following aspects:

- OpenShift configurations: Verify OpenShift configuration meets the standard and best practices.
- Security: It examines the security measures in place, such as authentication and authorization configurations.
- Application Probes: It tests the health and readiness probes of deployed applications to ensure they are functioning correctly.
- Resource Usage: It monitors resource consumption of OpenShift clusters, including CPU, memory, and storage.

The purpose of this application is to provide administrators and developers with an overview of OpenShift's health and functionality, helping them identify potential issues and ensure the smooth operation of their OpenShift environment.
*/

package openshift

import (
	"bufio"
	"context"
	"fmt"
	"github.com/fatih/color"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"os"
	"path/filepath"
	"strings"
)

var serviceMonitorFound int

func serviceMonitor() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	// Create a dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create dynamic client: %v\n", err)
		os.Exit(1)
	}
	// Retrieve all ServiceMonitors in all namespaces
	groupVersion := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	}
	serviceMonitorList, err := client.Resource(groupVersion).Namespace("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get ServiceMonitors: %v\n", err)
		os.Exit(1)
	}

	// Exclude ServiceMonitors with specific namespace prefixes
	excludedPrefixes := []string{"default", "openshift", "kube", "open"}
	var filteredServiceMonitors []unstructured.Unstructured

	for _, sm := range serviceMonitorList.Items {
		namespace := sm.GetNamespace()

		namespacePrefix := strings.Split(namespace, "-")[0]
		if !contains(excludedPrefixes, namespacePrefix) {
			filteredServiceMonitors = append(filteredServiceMonitors, sm)
		}
	}

	// Check if any ServiceMonitors are found
	serviceMonitorFound = len(filteredServiceMonitors)

	if serviceMonitorFound == 0 {
		color.Red("ServiceMonitors not configured\t\t\t\tFAILED")

	} else {
		// Print the filtered ServiceMonitors
		//for _, sm := range filteredServiceMonitors {
		//	namespace := sm.GetNamespace()
		//	name := sm.GetName()
		//	fmt.Printf("Namespace: %s, Name: %s\n", namespace, name)
		//}
		color.Green("ServiceMonitors is configured\t\t\t\tPASSED")
	}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	inputFile, err := os.Open(filepath.Join(dir, "resources/healthcheck-body.adoc"))
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	// Open the output file
	outputFile, err := os.OpenFile("resources/healthcheck-body.adoc", os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	// Create a scanner to read the input file
	scanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(serviceMonitorProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func serviceMonitorProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Application specific metrics are monitored on OpenShift>>") {
		// Render the change status
		if serviceMonitorFound == 0 {
			return line + "\n\n| ServiceMonitors not configured \n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| ServiceMonitors is configured \n\n" + GetKeyChanges("advisory")
		}
	}

	if strings.HasPrefix(line, "== Application specific metrics are monitored on OpenShift") {

		version, _ := getOpenShiftVersion()

		// Render the change status
		if serviceMonitorFound == 0 {
			return line + "\n\n" + GetChanges("recommended") +
				"\n\n**Observation**\n\nInsufficient tooling and processes are in place to ensure that applications deployed to OpenShift are performing as expected. \n\n" +
				"**Recommendation**\n\nOpenShift can monitor that application pods are started, and pass basic checks." +
				" Additional consideration should be given to capturing application specific metrics to identify performance and reliability issues for applications deployed to OCP. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/index#managing-metrics[Managing Metrics]\n" +
				"* https://www.dynatrace.com/technologies/openshift-monitoring/[Monitor applications deployed to OpenShift using 3rd party tools such as Dynatrace]\n" +
				"* https://www.appdynamics.com/solutions/openshift[Monitor applications deployed to OpenShift using 3rd party tools such as AppDynamics]\n" +
				"* https://newrelic.com/blog/how-to-relic/red-hat-openshift-monitor[Monitor applications deployed to OpenShift using 3rd party tools such as NewRelic]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") +
				"\n\n**Observation**\n\nSome tools use to monitor application performance. \n\n" +
				"**Recommendation**\n\nNone.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"html-single/monitoring/index#managing-metrics[Managing Metrics]\n" +
				"* https://www.dynatrace.com/technologies/openshift-monitoring/[Monitor applications deployed to OpenShift using 3rd party tools such as Dynatrace]\n" +
				"* https://www.appdynamics.com/solutions/openshift[Monitor applications deployed to OpenShift using 3rd party tools such as AppDynamics]\n" +
				"* https://newrelic.com/blog/how-to-relic/red-hat-openshift-monitor[Monitor applications deployed to OpenShift using 3rd party tools such as NewRelic]\n"

		}

	}
	return line + "\n"
}

// Helper function to check if a string is present in a slice
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
