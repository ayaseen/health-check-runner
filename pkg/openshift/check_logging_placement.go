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
	"fmt"
	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/util/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var appropriate bool

// Pod represents the basic information of an Elasticsearch pod.
type Pod struct {
	Name          string
	NodeName      string
	NodeSelectors map[string]string
}

// Check Logging

func checkLoggingPlacement() {

	pods, err := getElasticsearchPods()
	if err != nil {
		log.Fatalf("Error retrieving Elasticsearch pods: %v", err)
	}

	loggingConfigure, _ := checkLoggingConfigure()

	// Check if Elasticsearch pods are scheduled on appropriate nodes
	appropriate = true
	if loggingConfigure != "" {
		for _, pod := range pods {
			if !isScheduledOnInfraNode(pod) {
				appropriate = false
			}
		}
		if appropriate {
			color.Green("Elasticsearch pods are scheduled on appropriate nodes\tPASSED")
		} else {
			color.Red("Elasticsearch pods are scheduled on appropriate nodes\tFAILED")
		}
	} else {
		color.HiCyan("Elasticsearch pods are scheduled on appropriate nodes\tSKIPPED")
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
	fileScanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for fileScanner.Scan() {
		line := fileScanner.Text()
		output.WriteString(LoggingPlacementProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func LoggingPlacementProcess(line string) string {

	if strings.HasPrefix(line, "<<OpenShift logging Elasticsearch pods are scheduled on appropriate nodes>>") {
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {
			if appropriate {
				return line + "\n\n| OpenShift logging Elasticsearch pods not scheduled on Infra nodes \n\n" + GetKeyChanges("recommended")
			} else {
				return line + "\n\n| OpenShift logging Elasticsearch pods are scheduled on Infra nodes \n\n" + GetKeyChanges("nochange")
			}
		} else {
			return line + "\n\n|Not Applicable" + GetKeyChanges("na") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== OpenShift logging Elasticsearch pods are scheduled on appropriate nodes") {

		version, _ := getOpenShiftVersion()
		loggingConfigure, _ := checkLoggingConfigure()

		if loggingConfigure != "" {

			if appropriate {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nOpenShift logging Elasticsearch pods are scheduled on Infra nodes.\n\n" +
					"**Recommendation**\n\nNone\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#infrastructure-moving-logging_cluster-logging-moving[Moving Logging to Infra nodes]\n"
			} else {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nThere is no alternative log aggregation configured for long term audit. \n\n" +
					"**Recommendation**\n\nYou need to logging forwarder to aggregate all the logs from your OpenShift Container Platform cluster," +
					" such as node system audit logs, application container logs, and infrastructure logs. \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/logging/index#infrastructure-moving-logging_cluster-logging-moving[Moving Logging to Infra nodes]\n"

			}
		} else {

			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\nOpenShift Logging is not installed or configured. \n\n" +
				"**Recommendation**\n\nYou need to deploy the logging subsystem to aggregate all the logs from your OpenShift Container Platform cluster," +
				" such as node system audit logs, application container logs, and infrastructure logs. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging-deploying[Installing OpenShift Logging]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/configuring-your-logging-deployment[Cluster Logging custom resource]\n" +
				"* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/logging/cluster-logging[Understanding Red Hat OpenShift Logging]\n"
		}

	}

	return line + "\n"
}

// getElasticsearchPods retrieves a list of Elasticsearch pods in the OpenShift logging namespace.
func getElasticsearchPods() ([]Pod, error) {
	cmd := exec.Command("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	pods, err := parseElasticsearchPods(output)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

// parseElasticsearchPods parses the JSON output and extracts the relevant pod information.
func parseElasticsearchPods(jsonOutput []byte) ([]Pod, error) {
	// Parse the JSON data into a structured form
	var data map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &data); err != nil {
		return nil, err
	}

	// Extract the pod information
	pods := make([]Pod, 0)
	items, ok := data["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to extract 'items' from JSON")
	}
	for _, item := range items {
		itemData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		metadata, ok := itemData["metadata"].(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := metadata["name"].(string)
		if !ok {
			continue
		}

		spec, ok := itemData["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		nodeName, ok := spec["nodeName"].(string)
		if !ok {
			continue
		}

		nodeSelectors := make(map[string]string)
		if nodeSelector, ok := spec["nodeSelector"].(map[string]interface{}); ok {
			for key, value := range nodeSelector {
				if strValue, ok := value.(string); ok {
					nodeSelectors[key] = strValue
				}
			}
		}

		pod := Pod{
			Name:          name,
			NodeName:      nodeName,
			NodeSelectors: nodeSelectors,
		}

		pods = append(pods, pod)
	}

	return pods, nil
}

// isScheduledOnInfraNode checks if a pod is scheduled on a node with "node-role.kubernetes.io/infra: ”" nodeSelector.
func isScheduledOnInfraNode(pod Pod) bool {
	nodeRole, ok := pod.NodeSelectors["node-role.kubernetes.io/infra"]
	return ok && nodeRole == ""
}
